import Fastify from "fastify";

const app = Fastify({ logger: true });

app.get("/health", async () => ({ ok: true, service: "cuckoo-ai-service" }));

app.post<{
  Body: { prompt?: string; model?: string; metadata?: Record<string, unknown> };
}>("/completion", async (request) => {
  return {
    text: "[stub] AI completion is not enabled for the MVP.",
    model: request.body?.model ?? "stub",
    usage: { inputTokens: 0, outputTokens: 0 },
  };
});

const port = Number(process.env.PORT ?? 18787);
const host = process.env.HOST ?? "0.0.0.0";

app.listen({ port, host }).catch((err) => {
  app.log.error(err);
  process.exit(1);
});
