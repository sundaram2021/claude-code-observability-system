import dotenv from 'dotenv';
dotenv.config({ path: '.env.local' });
import './instrumentation.ts';

import express from 'express';
import path from 'path';
import { fileURLToPath } from 'url';
import { memorySpanExporter, memoryLogStore } from './instrumentation.ts';
import { chatWithTools } from './services/agent.ts';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const app = express();
app.use(express.json());
app.use(express.static(path.join(__dirname, '..', 'public')));

app.post('/api/chat', async (req, res) => {
  try {
    const { message } = req.body;
    const conversation = await chatWithTools(message);
    res.json({ success: true, conversation });
  } catch (error: any) {
    res.status(500).json({ success: false, error: error.message });
  }
});

app.get('/api/telemetry', (_req, res) => {
  const rawSpans = memorySpanExporter.getFinishedSpans();

  const spans = rawSpans.map(span => {
    const ctx = span.spanContext();
    return {
      id: ctx.spanId,
      traceId: ctx.traceId,
      parentId: span.parentSpanContext?.spanId,
      name: span.name,
      startTime: span.startTime,
      endTime: span.endTime,
      durationMs: span.duration[0] * 1000 + span.duration[1] / 1e6,
      attributes: span.attributes,
      events: span.events,
    };
  });

  res.json({
    spans,
    logs: memoryLogStore,
  });
});

app.listen(3000, () => console.log('Server running on http://localhost:3000'));