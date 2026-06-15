import { trace } from '@opentelemetry/api';
import { memoryLogStore } from '../instrumentation.ts';

export const logger = {
  info: (message: string) => log('INFO', message),
  error: (message: string, error?: any) => log('ERROR', `${message} ${error?.message || ''}`),
};

function log(level: string, message: string) {
  const activeSpan = trace.getActiveSpan();
  const traceId = activeSpan?.spanContext().traceId;

  memoryLogStore.push({
    timestamp: new Date().toISOString(),
    level,
    message,
    traceId,
  });

  console.log(`[${level}] ${message} ${traceId ? `(trace_id: ${traceId})` : ''}`);
}