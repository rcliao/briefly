# Design: Banner Image Generation for the Weekly Digest

**Status:** Implemented 2026-08-05 (review decisions: default on; pixel-art style; gemini-3.1-flash-image; interactive 1-of-3 prompt pick, single prompt saved)

## Problem

Every week the digest gets shared on LinkedIn with a banner image. Today that means two manual steps outside the tool: asking Claude for image prompt ideas, then pasting a prompt into Google Chat, generating, and downloading the image. The digest pipeline knows everything the prompt needs (title, topics, the week's throughline) — it should carry the banner most or all of the way.

Verified premise: the same `GEMINI_API_KEY` the digest already uses can generate images directly (`gemini-3.1-flash-image` supports `generateContent`). So we can go beyond suggesting prompts — the CLI can produce the finished PNG.

## Target workflow

```
1. paste URLs into input doc                        (unchanged)
2. briefly digest from-file input/weekly.md         (digest + banner.png + prompts come out)
3. paste digest into LinkedIn, upload banner.png,   (unchanged — post text stays yours)
   schedule
```

Steps 3–4 of the old workflow (prompt-asking, Google Chat round-trip) disappear. When the generated image isn't right, a saved prompts file keeps the manual Google Chat path one paste away.

## Design

### 1. Where banner generation runs

Generate during the digest run, on by default, with two escape hatches:

- `--no-banner` — skip it (adds ~5–10s and ~1–4¢ per run otherwise)
- `briefly banner <digest-file>` — standalone regeneration: re-reads the (possibly hand-edited) digest, writes new prompts + image without re-running the pipeline. This is the retry loop when the first image misses.

Alternative considered: *only* the standalone subcommand (digest stays image-free). Rejected as default because it re-adds a manual step to the happy path; but the subcommand exists anyway for retries, so both shapes are available.

### 2. What gets produced

For `digests/digest_2026-08-05.md`:

- `digests/banner_2026-08-05.png` — the generated image
- `digests/banner_2026-08-05.prompts.txt` — 3 prompt variants (the used one first), ready to paste into Google Chat if the PNG disappoints
- The digest `.md` itself stays untouched — it must remain pure pasteable text

CLI output prints the chosen prompt so you see what drove the image.

### 3. How the prompt is written

A second small LLM call after the editorial pass (not folded into the editorial call — image prompting has its own craft and shouldn't compete with digest quality in one response). Input: digest title, executive summary, topic names. Output: 3 prompt variants.

Prompt-writing rules baked into the system prompt:

- **Series identity over novelty.** A recognizable weekly look beats a fresh style every week (same principle as the digest format itself). The style half of the prompt is fixed configuration; the LLM only writes the *subject* half from the week's content.
- **No text in the image.** Image models garble words, and LinkedIn shows the post title anyway. Prompts explicitly forbid lettering.
- **Concrete visual metaphor, not abstraction.** "A robotic arm passing a glowing envelope across a chasm between two server towers" beats "an abstract representation of AI communication."

### 4. Style configuration

`.briefly.yaml` gains:

```yaml
banner:
  enabled: true
  model: "gemini-3.1-flash-image"
  style: "flat vector illustration, dark navy background, electric blue and
          warm orange accents, isometric perspective, clean geometric shapes,
          no text or lettering"
  aspect_ratio: "16:9"   # closest supported ratio to LinkedIn's 1.91:1
```

Defaults ship in code; the style string is the knob for evolving the series look without touching Go.

### 5. Failure behavior

Image generation failing (safety filter, quota, API hiccup) must never fail the digest. On failure: prompts file is still written, CLI prints the first prompt with a note to use the Google Chat path. Same graceful-degradation contract as the rest of the pipeline.

## Non-goals

- **Writing the LinkedIn post.** You write those yourself now, deliberately. The tool stops at digest + banner.
- **Multiple generated images to choose from.** One image, three prompts. If the image misses, `briefly banner` regenerates (optionally with `--prompt-index 2` or `--prompt "custom"`). Generating 3 images per run triples cost for a choice you'd rarely exercise — revisit if regeneration turns out to be frequent.
- **Image editing/refinement loops.** Out of scope; the prompts file is the escape hatch.

## Open questions for review

1. **Default on or off?** Proposed: on (`--no-banner` to skip). If you usually only share some weeks, default-off with `--banner` might fit better.
2. **The style direction above is a placeholder.** What look do you actually want for the series? (Existing banners you liked from the Google Chat era would be the best reference.)
3. **Model choice:** `gemini-3.1-flash-image` (fast/cheap) vs `gemini-3-pro-image` (higher quality, slower, pricier). Proposed: flash-image; pro is a config change away.
4. **Prompt variants in the sidecar:** is 3 the right number, or is 1 enough?

## Implementation sketch (after design settles)

- `internal/llm`: add `GenerateImage(ctx, model, prompt, aspectRatio) ([]byte, error)` using `genai` `ResponseModalities: [IMAGE, TEXT]`
- `cmd/handlers/banner.go`: prompt-writing call + image call + file writes; invoked from both the digest pipeline and the `banner` subcommand
- Config: `banner` section with defaults
- Tests: prompt-builder unit test; image call behind an interface for testability
