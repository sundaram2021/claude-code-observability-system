import { trace } from '@opentelemetry/api';
import Anthropic from '@anthropic-ai/sdk';

const tracer = trace.getTracer('chat-agent');

let _anthropic: Anthropic;
function getClient() {
  if (!_anthropic) _anthropic = new Anthropic({ apiKey: process.env.ANTHROPIC_API_KEY });
  return _anthropic;
}

async function executeToolWithTracing(toolName: string, args: any) {
  return tracer.startActiveSpan(`tool_execution: ${toolName}`, async (span) => {
    span.setAttribute('gen_ai.tool.name', toolName);
    span.setAttribute('gen_ai.tool.arguments', JSON.stringify(args));

    try {
      let result: string;
      if (toolName === 'get_date') {
        result = new Date().toISOString();
      } else if (toolName === 'calculator') {
        result = String(Function('"use strict"; return (' + args.expression + ')')());
      } else {
        result = `Unknown tool: ${toolName}`;
      }

      span.setAttribute('gen_ai.tool.result', result);
      return result;
    } catch (error) {
      span.recordException(error as Error);
      throw error;
    } finally {
      span.end();
    }
  });
}

export async function chatWithTools(userMessage: string) {
  return tracer.startActiveSpan('chat.request', async (span) => {
    let messages: Anthropic.MessageParam[] = [{ role: 'user', content: userMessage }];
    let isDone = false;

    while (!isDone) {
      const response = await getClient().messages.create({
        model: 'claude-haiku-4-5-20251001',
        max_tokens: 1024,
        tools: [
          {
            name: 'get_date',
            description: 'Gets the current date and time',
            input_schema: { type: 'object' as const, properties: {} },
          },
          {
            name: 'calculator',
            description: 'Evaluates math expressions',
            input_schema: {
              type: 'object' as const,
              properties: { expression: { type: 'string' } },
              required: ['expression'],
            },
          },
        ],
        messages,
      });

      messages.push({ role: 'assistant', content: response.content });

      if (response.stop_reason === 'tool_use') {
        const toolUses = response.content.filter(
          (block): block is Anthropic.ToolUseBlock => block.type === 'tool_use'
        );

        const toolResults: Anthropic.ToolResultBlockParam[] = [];
        for (const tool of toolUses) {
          const result = await executeToolWithTracing(tool.name, tool.input);
          toolResults.push({
            type: 'tool_result',
            tool_use_id: tool.id,
            content: result,
          });
        }

        messages.push({ role: 'user', content: toolResults });
      } else {
        isDone = true;
      }
    }
    span.end();
    return messages;
  });
}