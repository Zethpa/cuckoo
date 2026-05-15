import Fastify from "fastify";

type JudgeRequest = {
  theme?: string;
  openingSentence?: string;
  previousStory?: string[];
  text?: string;
  units?: number;
  maxUnits?: number;
  timeTakenMs?: number;
  timeLimitMs?: number;
  judgePersona?: string;
  provider?: {
    baseUrl?: string;
    apiKey?: string;
    model?: string;
    apiStyle?: "responses" | "chat_completions";
  };
};

type JudgeResult = {
  fluency: number;
  explain: string;
  model: string;
  language: string;
};

type AIConfig = {
  enabled: boolean;
  apiKey: string;
  baseUrl: string;
  model: string;
  apiStyle: "responses" | "chat_completions";
  judgePersona: string;
};

const app = Fastify({ logger: true });

app.get("/health", async () => ({
  ok: true,
  service: "cuckoo-ai-service",
  aiEnabled: loadAIConfig().enabled,
}));

app.post<{
  Body: { prompt?: string; model?: string; metadata?: Record<string, unknown> };
}>("/completion", async (request) => {
  return {
    text: "[stub] AI completion is not enabled for the MVP.",
    model: request.body?.model ?? "stub",
    usage: { inputTokens: 0, outputTokens: 0 },
  };
});

app.post<{ Body: JudgeRequest }>("/judge", async (request) => {
  const cfg = loadAIConfig(request.body);
  if (!cfg.enabled) {
    return localJudge("AI judge is disabled or OPENAI_API_KEY is not configured.");
  }
  try {
    return cfg.apiStyle === "chat_completions"
      ? await judgeWithChatCompletions(cfg, request.body)
      : await judgeWithResponses(cfg, request.body);
  } catch (err) {
    request.log.warn({ err }, "AI judge failed; falling back to local placeholder");
    return localJudge("AI judge failed; local fallback score was used.");
  }
});

function loadAIConfig(input?: JudgeRequest): AIConfig {
  const provider = input?.provider;
  const apiKey = provider?.apiKey ?? process.env.OPENAI_API_KEY ?? "";
  const apiStyle = normalizeAPIStyle(provider?.apiStyle ?? process.env.OPENAI_API_STYLE);
  return {
    enabled: process.env.AI_JUDGE_ENABLED !== "false" && apiKey.length > 0,
    apiKey,
    baseUrl: stripTrailingSlash(provider?.baseUrl ?? process.env.OPENAI_BASE_URL ?? "https://api.openai.com/v1"),
    model: provider?.model ?? process.env.OPENAI_MODEL ?? "gpt-4.1-mini",
    apiStyle,
    judgePersona: input?.judgePersona ?? process.env.AI_JUDGE_PERSONA ?? defaultJudgePersona,
  };
}

async function judgeWithResponses(cfg: AIConfig, input: JudgeRequest): Promise<JudgeResult> {
  const res = await fetch(`${cfg.baseUrl}/responses`, {
    method: "POST",
    headers: openAIHeaders(cfg),
    body: JSON.stringify({
      model: cfg.model,
      input: [
        { role: "system", content: [{ type: "input_text", text: cfg.judgePersona }] },
        { role: "user", content: [{ type: "input_text", text: buildJudgeInput(input) }] },
      ],
      text: {
        format: {
          type: "json_schema",
          name: "cuckoo_judge_result",
          strict: true,
          schema: judgeJSONSchema,
        },
      },
    }),
  });
  const payload = await readJSON(res);
  const text = extractResponsesText(payload);
  return normalizeJudgeResult(JSON.parse(text), cfg.model);
}

async function judgeWithChatCompletions(cfg: AIConfig, input: JudgeRequest): Promise<JudgeResult> {
  const res = await fetch(`${cfg.baseUrl}/chat/completions`, {
    method: "POST",
    headers: openAIHeaders(cfg),
    body: JSON.stringify({
      model: cfg.model,
      messages: [
        { role: "system", content: cfg.judgePersona },
        { role: "user", content: buildJudgeInput(input) },
      ],
      response_format: {
        type: "json_schema",
        json_schema: {
          name: "cuckoo_judge_result",
          strict: true,
          schema: judgeJSONSchema,
        },
      },
    }),
  });
  const payload = await readJSON(res) as { choices?: Array<{ message?: { content?: string } }> };
  const text = payload?.choices?.[0]?.message?.content;
  if (typeof text !== "string") {
    throw new Error("Chat Completions response did not contain message content");
  }
  return normalizeJudgeResult(JSON.parse(text), cfg.model);
}

function buildJudgeInput(input: JudgeRequest): string {
  const previousStory = input.previousStory?.filter(Boolean).join("\n") || input.openingSentence || "";
  return JSON.stringify({
    theme: input.theme ?? "",
    openingSentence: input.openingSentence ?? "",
    previousStory,
    continuation: input.text ?? "",
    units: input.units ?? 0,
    maxUnits: input.maxUnits ?? 0,
    timeTakenMs: input.timeTakenMs ?? 0,
    timeLimitMs: input.timeLimitMs ?? 0,
  });
}

function normalizeJudgeResult(value: unknown, fallbackModel: string): JudgeResult {
  const data = value as Partial<JudgeResult>;
  const fluency = Math.max(0, Math.min(20, Math.round(Number(data.fluency ?? 20))));
  const explain = typeof data.explain === "string" && data.explain.trim() !== ""
    ? data.explain.trim()
    : "AI judge returned a fluency score.";
  const language = typeof data.language === "string" && data.language.trim() !== ""
    ? data.language.trim()
    : "auto";
  const model = typeof data.model === "string" && data.model.trim() !== ""
    ? data.model.trim()
    : fallbackModel;
  return { fluency, explain, model, language };
}

function extractResponsesText(payload: unknown): string {
  const data = payload as {
    output_text?: string;
    output?: Array<{ content?: Array<{ type?: string; text?: string }> }>;
  };
  if (typeof data.output_text === "string") {
    return data.output_text;
  }
  for (const item of data.output ?? []) {
    for (const content of item.content ?? []) {
      if (content.type === "output_text" && typeof content.text === "string") {
        return content.text;
      }
    }
  }
  throw new Error("Responses API result did not contain output text");
}

async function readJSON(res: Response): Promise<unknown> {
  const text = await res.text();
  if (!res.ok) {
    throw new Error(`AI provider returned ${res.status}: ${text.slice(0, 500)}`);
  }
  return JSON.parse(text);
}

function openAIHeaders(cfg: AIConfig): Record<string, string> {
  return {
    "Content-Type": "application/json",
    Authorization: `Bearer ${cfg.apiKey}`,
  };
}

function normalizeAPIStyle(value?: string): "responses" | "chat_completions" {
  return value === "chat_completions" ? "chat_completions" : "responses";
}

function stripTrailingSlash(value: string): string {
  return value.replace(/\/+$/, "");
}

function localJudge(explain: string): JudgeResult {
  return {
    fluency: 20,
    explain,
    model: "local-fallback",
    language: "auto",
  };
}

const judgeJSONSchema = {
  type: "object",
  additionalProperties: false,
  required: ["fluency", "explain", "language", "model"],
  properties: {
    fluency: {
      type: "integer",
      minimum: 0,
      maximum: 20,
      description: "Fluency and narrative-fit score from 0 to 20.",
    },
    explain: {
      type: "string",
      description: "Short explanation in the automatically selected evaluation language.",
    },
    language: {
      type: "string",
      description: "BCP-47-ish language label or short language name used for explain.",
    },
    model: {
      type: "string",
      description: "Model name used by the judge.",
    },
  },
};

const defaultJudgePersona = [
  "You are Cuckoo's story-continuation judge.",
  "Evaluate only the submitted continuation against the given theme and previous story context.",
  "Return a fair fluency score from 0 to 20, where 20 means vivid, coherent, context-aware, and naturally continuing the story; 10 means understandable but plain or weakly connected; 0 means incoherent, empty, or unrelated.",
  "Do not punish the player for server-side timing or word-count compliance; those are scored separately.",
  "Automatically choose the language of the explanation based on the story context and continuation. If the story is mostly Chinese, explain in Chinese. If mostly English, explain in English. If mixed, use the dominant language or a concise bilingual note.",
  "Keep explain to one or two short sentences.",
  "Output only JSON that matches the provided schema.",
].join("\n");

const port = Number(process.env.PORT ?? 18787);
const host = process.env.HOST ?? "0.0.0.0";

app.listen({ port, host }).catch((err) => {
  app.log.error(err);
  process.exit(1);
});
