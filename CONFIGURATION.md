# Briefly Configuration Guide

This document explains how to configure Briefly for optimal performance with various AI and search providers.

## Configuration Methods

Briefly supports multiple configuration methods, listed in order of precedence:

1. **Command-line flags** (highest priority)
2. **Environment variables** (via `.env` file or system environment)
3. **YAML configuration file** (`.briefly.yaml`)
4. **Default values** (lowest priority)

## Quick Start

### Minimal Setup

1. **Copy configuration templates:**
   ```bash
   cp .env.example .env
   cp .briefly.yaml.example .briefly.yaml
   ```

2. **Set your Gemini API key:**
   ```bash
   # In .env file
   GEMINI_API_KEY=your-gemini-api-key-here
   ```

3. **Test the setup:**
   ```bash
   briefly digest input/test-links.md
   ```

### Adding Search Providers

For research features, configure at least one search provider:

```bash
# Google Custom Search (Recommended)
GOOGLE_CSE_API_KEY=your-google-api-key
GOOGLE_CSE_ID=your-search-engine-id

# Or SerpAPI (Premium)
SERPAPI_KEY=your-serpapi-key
```

## Configuration Files

### Environment Variables (.env)

The `.env` file is ideal for sensitive information like API keys:

```env
# Required
GEMINI_API_KEY=your-gemini-api-key

# Search providers (choose one or more)
GOOGLE_CSE_API_KEY=your-google-key
GOOGLE_CSE_ID=your-search-engine-id
SERPAPI_KEY=your-serpapi-key

# Optional integrations
SLACK_WEBHOOK_URL=https://hooks.slack.com/...
OPENAI_API_KEY=your-openai-key
```

### YAML Configuration (.briefly.yaml)

The YAML file is ideal for non-sensitive settings and defaults:

```yaml
gemini:
  model: "gemini-2.5-flash-preview-05-20"

output:
  directory: "my-digests"

research:
  default_provider: "google"
  max_results_per_query: 15

deep_research:
  max_sources: 30
  since: "14d"
```

## API Keys and Setup

### Google Gemini API

**Required for**: All AI features (summarization, research, synthesis)

**Setup:**
1. Visit [Google AI Studio](https://makersuite.google.com/app/apikey)
2. Create a new API key
3. Add to `.env`: `GEMINI_API_KEY=your-key-here`

**Models Available:**
- `gemini-1.5-pro`: Best quality, slower, higher cost
- `gemini-1.5-flash`: Good balance of speed and quality
- `gemini-2.5-flash-preview-05-20`: Latest Gemini 2.5 Flash Preview (default) - fastest and most cost-effective

### Google Custom Search

**Required for**: High-quality web search in research features

**Setup:**
1. Create a Custom Search Engine at [Google CSE](https://cse.google.com/cse/)
2. Get API key from [Google Cloud Console](https://console.cloud.google.com/apis/credentials)
3. Add to `.env`:
   ```env
   GOOGLE_CSE_API_KEY=your-api-key
   GOOGLE_CSE_ID=your-search-engine-id
   ```

**Usage:**
```bash
briefly research "AI trends" --provider google
briefly deep-research "machine learning" --search-provider google
```

### SerpAPI

**Required for**: Premium search results with advanced features

**Setup:**
1. Sign up at [SerpAPI](https://serpapi.com/dashboard)
2. Get your API key from the dashboard
3. Add to `.env`: `SERPAPI_KEY=your-serpapi-key`

**Usage:**
```bash
briefly research "AI trends" --provider serpapi
briefly deep-research "machine learning" --search-provider serpapi
```

## Search Provider Comparison

| Provider | Cost | Quality | Rate Limits | Setup Difficulty |
|----------|------|---------|-------------|------------------|
| DuckDuckGo | Free | Good | 2s between requests | None |
| Google CSE | $5/1000 queries | Excellent | 100 queries/day free | Medium |
| SerpAPI | $50/5000 queries | Excellent | Generous | Easy |
| Mock | Free | N/A (testing) | None | None |

**Recommendation:** Start with DuckDuckGo for testing, upgrade to Google CSE for production use.

## Multi-Channel Output

### Slack Integration

**Setup:**
1. Create incoming webhook in Slack
2. Add to `.env`: `SLACK_WEBHOOK_URL=https://hooks.slack.com/...`

**Usage:**
```bash
briefly send-slack input/links.md --format summary
```

### Discord Integration

**Setup:**
1. Create webhook in Discord channel settings
2. Add to `.env`: `DISCORD_WEBHOOK_URL=https://discord.com/api/webhooks/...`

**Usage:**
```bash
briefly send-discord input/links.md --format highlights
```

## Text-to-Speech

### OpenAI TTS

**Setup:**
1. Get API key from [OpenAI Platform](https://platform.openai.com/api-keys)
2. Add to `.env`: `OPENAI_API_KEY=your-openai-key`

**Usage:**
```bash
briefly generate-tts input/digest.md --provider openai --voice alloy
```

### ElevenLabs

**Setup:**
1. Get API key from [ElevenLabs](https://elevenlabs.io/speech-synthesis)
2. Add to `.env`: `ELEVENLABS_API_KEY=your-elevenlabs-key`

**Usage:**
```bash
briefly generate-tts input/digest.md --provider elevenlabs --voice Rachel
```

## Configuration Precedence Examples

### Command Line Override
```bash
# Override YAML/env config for one command
briefly digest input/links.md --format newsletter --output /tmp/digest
```

### Environment Variable Override
```bash
# Override YAML config for current session
GEMINI_MODEL=gemini-1.5-pro briefly digest input/links.md
```

### YAML Configuration
```yaml
# Set defaults in .briefly.yaml
gemini:
  model: "gemini-1.5-flash"
output:
  directory: "my-digests"
```

## Troubleshooting

### Common Issues

**"API key not valid"**
- Check your API key is correct
- Ensure no extra spaces or characters
- Verify the service is enabled in your cloud console

**"Search provider error"**
- Verify API keys are set correctly
- Check rate limits haven't been exceeded
- Try a different search provider

**"No results found"**
- Try different search terms
- Check if time filters are too restrictive
- Verify network connectivity

### Debug Mode

Enable detailed logging:

```bash
# Set debug mode
DEBUG=1 briefly research "AI trends"

# Or in YAML
logging:
  level: "debug"
```

### Testing Configuration

Use mock providers to test without API costs:

```bash
# Test with mock providers
briefly research "test query" --provider mock
briefly deep-research "test topic" --search-provider mock
```

## Security Best Practices

1. **Never commit API keys to version control**
2. **Use `.env` files for sensitive data**
3. **Set appropriate file permissions**: `chmod 600 .env`
4. **Use different API keys for development/production**
5. **Regularly rotate API keys**
6. **Monitor API usage for unexpected costs**

## Performance Optimization

### Cache Configuration

```yaml
cache:
  directory: ".briefly-cache"
  article_ttl: "24h"     # Keep articles cached for 24 hours
  summary_ttl: "168h"    # Keep summaries cached for 7 days
```

### Rate Limiting

```yaml
# In YAML or via environment variables
search:
  rate_limit_ms: 1000    # 1 second between search requests
```

### Cost Management

```yaml
features:
  cost_tracking: true    # Track API usage costs
```

Monitor usage:
```bash
briefly cost-estimate input/large-file.md  # Estimate before processing
```

## Example Configurations

### Development Setup
```yaml
# .briefly.yaml for development
gemini:
  model: "gemini-2.5-flash-preview-05-20"  # Fastest, most cost-effective
logging:
  level: "debug"
features:
  cost_tracking: true
```

### Production Setup
```yaml
# .briefly.yaml for production
gemini:
  model: "gemini-1.5-pro"    # Higher quality
research:
  default_provider: "google"  # Best search quality
cache:
  article_ttl: "168h"        # Longer cache for efficiency
logging:
  level: "info"
  format: "json"
```

### Minimal Setup
```env
# .env - minimal configuration
GEMINI_API_KEY=your-gemini-key
GOOGLE_CSE_API_KEY=your-google-key
GOOGLE_CSE_ID=your-search-engine-id
```