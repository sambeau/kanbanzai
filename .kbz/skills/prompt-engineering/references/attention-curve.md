# Attention Curve Deep Dive

> Reference file for `prompt-engineering` skill. Loaded on demand when
> the agent needs deeper understanding of U-shaped attention.

## The Finding

Liu et al. ("Lost in the Middle", 2024) demonstrated that language models
perform significantly worse at retrieving and using information placed in
the middle of their context window compared to the beginning or end.

The accuracy degradation is substantial: **30%+ drop** for critical
information in the middle of context. This is not a subtle effect — it's
the difference between correct and incorrect output on many tasks.

## The Mechanism

Wu et al. (MIT, 2025) identified the architectural causes:

**Causal masking** means each token can only attend to preceding tokens
(not future ones). This creates an asymmetry: early tokens are attended
to by everything that follows, while late tokens are attended to by very
few subsequent tokens. But late tokens benefit from **recency bias** — the
model's generation is most heavily influenced by the immediately preceding
context.

**RoPE (Rotary Position Embedding)** encodes position information through
rotation. The decay pattern it creates means that tokens in the middle
are neither close enough to the start (to benefit from early-position
attention) nor close enough to the end (to benefit from recency).

The result is the **U-shaped attention curve**:
- Beginning of context: HIGH attention (all subsequent tokens attend here)
- Middle of context: LOW attention (the "attention valley")
- End of context: HIGH attention (recency bias + generation influence)

## Anthropic's Confirmation

Anthropic's "Effective Context Engineering" (September 2025) independently
confirmed this pattern in production systems. Their concept of the
**attention budget** formalises it: every token competes with every other
token. Tokens in the middle lose this competition to tokens at the edges.

The optimal utilisation zone is **15–40%** of the context window. Below
~10%, hallucination risk increases (insufficient grounding). Above ~60%,
attention dilution dominates and middle-content loss accelerates.

## Section Ordering Strategy

Based on these findings, the recommended section ordering for any prompt
or skill document is:

| Position | Attention | What to put here | Why |
|----------|-----------|------------------|-----|
| Top | HIGH | Identity + vocabulary payload | Routing signal — determines which knowledge clusters activate |
| Near top | HIGH | Constraints + anti-patterns | Hard rules benefit from peak attention |
| Near top | HIGH | Checklist | Must be seen early to be used throughout |
| Middle | LOWER | Procedure | Numbered steps survive attention degradation better than prose |
| Near bottom | RISING | Output format + examples | Recency bias improves pattern matching against desired output |
| Bottom | HIGH | Retrieval anchors | End-of-context attention boost; benefits semantic indexing |

## What Makes the Middle Survivable

Not all middle content is equally degraded:

1. **Numbered lists** survive better than prose because the numbering
   structure provides positional scaffolding that partially compensates
   for attention dilution.

2. **Imperative verbs** at the start of each step survive better than
   passive descriptions — they act as mini attention anchors.

3. **IF/THEN branches** within steps survive better than long conditional
   paragraphs — the structure creates internal breaks that trigger fresh
   attention windows.

4. **Concrete nouns** (tool names, file paths, exact values) survive
   better than abstract concepts in the middle because they have fewer
   competing associations.

## Practical Implications

When you place something in the middle of a prompt:

- Assume the agent will give it ~30% less attention than content at the
  edges.
- If it's a critical constraint, move it to the top.
- If it's a pattern the agent should match against, move it near the
  bottom (recency bias).
- If it must stay in the middle, make it a numbered step with an
  imperative verb.

## Key Sources

- Liu et al., "Lost in the Middle: How Language Models Use Long Contexts"
  (2024)
- Wu et al., MIT Position Bias Analysis (2025)
- Anthropic, "Effective Context Engineering" (September 2025)
- Voyce, "XML/Markdown Comparative Study" (2025) — prompt format accounts
  for up to 40% performance variance
