import { NodeSDK } from '@opentelemetry/sdk-node';
import { InMemorySpanExporter, SimpleSpanProcessor } from '@opentelemetry/sdk-trace-node';
import { AnthropicInstrumentation } from '@arizeai/openinference-instrumentation-anthropic';
import Anthropic from '@anthropic-ai/sdk';

export const memorySpanExporter = new InMemorySpanExporter();

export const memoryLogStore: Array<{ timestamp: string; level: string; message: string; traceId?: string }> = [];

const anthropicInstrumentation = new AnthropicInstrumentation();
anthropicInstrumentation.manuallyInstrument(Anthropic);

const sdk = new NodeSDK({
  spanProcessor: new SimpleSpanProcessor(memorySpanExporter),
  instrumentations: [anthropicInstrumentation],
});

sdk.start();