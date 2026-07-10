-- Normalize historical Custom account protocol storage.
UPDATE accounts
SET
    extra = jsonb_set(COALESCE(extra, '{}'::jsonb), '{protocol}', to_jsonb(credentials->>'protocol'), true),
    credentials = COALESCE(credentials, '{}'::jsonb) - 'protocol',
    updated_at = NOW()
WHERE platform = 'custom'
  AND COALESCE(extra->>'protocol', '') = ''
  AND credentials->>'protocol' IN ('openai_responses', 'openai_chat_completions', 'openai_embeddings', 'anthropic_messages', 'gemini');
