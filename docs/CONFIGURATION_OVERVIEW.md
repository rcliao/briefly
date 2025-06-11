# Briefly Configuration System Overview

This document provides a comprehensive overview of Briefly's centralized configuration management system, making it easy to understand what configuration is needed across all modules.

## 🏗️ **Architecture**

### Centralized Configuration Module
- **Location**: `/internal/config/config.go`
- **Purpose**: Single source of truth for all application configuration
- **Benefits**: Easy to understand, maintain, and extend configuration across all modules

### Configuration Sources (in order of precedence)
1. **Command-line flags** (highest priority)
2. **Environment variables** (flexible naming support)
3. **YAML configuration file** (`.briefly.yaml`)
4. **Default values** (lowest priority)

## 📋 **Complete Configuration Reference**

### 🔧 **Application Settings**
```yaml
app:
  debug: false                    # Enable debug mode
  log_level: "info"              # Logging level: debug, info, warn, error
  data_dir: ".briefly-cache"     # Data directory for cache and storage
```

**Environment Variables**: `DEBUG`, `BRIEFLY_DEBUG`

### 🤖 **AI/LLM Configuration**
```yaml
ai:
  gemini:
    api_key: ""                  # Gemini API key (required)
    model: "gemini-2.5-flash-preview-05-20"  # AI model
    timeout: "30s"               # Request timeout
    max_tokens: 8192             # Maximum tokens per request
    temperature: 0.7             # Generation randomness (0.0-1.0)
    embedding_model: "text-embedding-004"  # Embedding model
  
  openai:
    api_key: ""                  # OpenAI API key (for DALL-E, TTS)
    model: "gpt-image-1"         # Image generation model
    base_url: "https://api.openai.com/v1"  # API base URL
    timeout: "30s"               # Request timeout
```

**Environment Variables**:
- Gemini: `GEMINI_API_KEY`, `GOOGLE_GEMINI_API_KEY`, `GOOGLE_AI_API_KEY`
- OpenAI: `OPENAI_API_KEY`

### 🔍 **Search Providers**
```yaml
search:
  default_provider: "duckduckgo" # Primary search provider
  max_results: 10                # Maximum results per query
  timeout: "15s"                 # Search timeout
  language: "en"                 # Search language
  
  providers:
    google:
      api_key: ""                # Google Custom Search API key
      search_id: ""              # Custom Search Engine ID
    
    serpapi:
      api_key: ""                # SerpAPI key
    
    duckduckgo:
      rate_limit: "1s"           # Rate limiting between requests
```

**Environment Variables**:
- Google: `GOOGLE_CUSTOM_SEARCH_API_KEY`, `GOOGLE_CSE_API_KEY`, `GOOGLE_SEARCH_API_KEY`
- Google ID: `GOOGLE_CUSTOM_SEARCH_ID`, `GOOGLE_CSE_ID`, `GOOGLE_SEARCH_ENGINE_ID`
- SerpAPI: `SERPAPI_API_KEY`, `SERPAPI_KEY`
- Provider: `SEARCH_PROVIDER`, `DEFAULT_SEARCH_PROVIDER`

### 📁 **Output & Templates**
```yaml
output:
  directory: "digests"           # Default output directory
  format: "standard"             # Default format: brief, standard, detailed, newsletter
  templates_dir: "templates"     # Template directory
```

### 🗂️ **Cache Management**
```yaml
cache:
  directory: ".briefly-cache"    # Cache directory
  database:
    timeout: "5s"                # Database operation timeout
  ttl:
    articles: "24h"              # Article cache duration
    summaries: "168h"            # Summary cache duration (7 days)
    digests: "720h"              # Digest cache duration (30 days)
    feeds: "1h"                  # Feed cache duration
```

### 🎨 **Visual/Banner Generation**
```yaml
visual:
  banners:
    default_style: "tech"        # Banner style: tech, minimalist, professional
    width: 1792                  # Image width
    height: 1024                 # Image height
    output_directory: "banners"  # Banner output directory
    quality: "high"              # Image quality
    format: "PNG"                # Image format
```

### 🔊 **Text-to-Speech**
```yaml
tts:
  default_provider: "openai"     # Default TTS provider
  default_voice: "alloy"         # Default voice
  default_speed: 1.0             # Speech speed
  output_directory: "audio"      # Audio output directory
  timeout: "60s"                 # TTS generation timeout
  
  providers:
    openai:
      api_key: ""                # OpenAI API key (shared with ai.openai)
      model: "tts-1"             # TTS model
    
    elevenlabs:
      api_key: ""                # ElevenLabs API key
    
    google:
      api_key: ""                # Google Cloud TTS API key
```

**Environment Variables**:
- ElevenLabs: `ELEVENLABS_API_KEY`, `ELEVEN_LABS_API_KEY`
- Google: `GOOGLE_TTS_API_KEY`, `GOOGLE_CLOUD_API_KEY`

### 💬 **Messaging Integration**
```yaml
messaging:
  default_format: "summary"      # Default message format
  timeout: "10s"                 # Request timeout
  
  slack:
    webhook_url: ""              # Slack webhook URL
    username: "Briefly"          # Bot username
    icon_emoji: ":newspaper:"    # Bot emoji
    default_channel: ""          # Default channel
  
  discord:
    webhook_url: ""              # Discord webhook URL
    username: "Briefly"          # Bot username
    avatar_url: ""               # Bot avatar URL
```

**Environment Variables**:
- Slack: `SLACK_WEBHOOK_URL`, `SLACK_WEBHOOK`
- Discord: `DISCORD_WEBHOOK_URL`, `DISCORD_WEBHOOK`

### 📧 **Email Configuration**
```yaml
email:
  smtp:
    host: ""                     # SMTP server host
    port: 587                    # SMTP port
    username: ""                 # SMTP username
    password: ""                 # SMTP password
    tls_enabled: true            # Enable TLS
  
  default_template: "default"    # Default email template
  from_address: ""               # From email address
  from_name: "Briefly"           # From name
```

**Environment Variables**:
- SMTP: `SMTP_HOST`, `EMAIL_SMTP_HOST`
- Auth: `SMTP_USERNAME`, `EMAIL_USERNAME`, `SMTP_PASSWORD`, `EMAIL_PASSWORD`

### 📰 **RSS/Feed Management**
```yaml
feeds:
  fetch_interval: "1h"           # How often to fetch feeds
  user_agent: "Briefly/1.0"     # User agent for requests
  timeout: "30s"                 # Request timeout
  max_items_per_feed: 50         # Maximum items per feed
  cleanup_interval: "24h"        # How often to clean old items
```

### 🔬 **Research Configuration**
```yaml
research:
  max_depth: 3                   # Maximum research depth
  max_queries: 5                 # Maximum queries per topic
  timeout: "60s"                 # Research timeout
  concurrent_searches: 3         # Parallel search limit
```

### 📝 **Logging System**
```yaml
logging:
  level: "info"                  # Log level: debug, info, warn, error
  format: "text"                 # Format: text, json
  output: "stdout"               # Output: stdout, stderr, or file path
  file_path: ""                  # Log file path (if output is file)
```

### 🖥️ **CLI Configuration**
```yaml
cli:
  editor: ""                     # Default editor (uses $EDITOR)
  interactive: false             # Enable interactive mode
  default_format: "standard"     # Default output format
  style_guide_path: ""           # Style guide file path
```

**Environment Variables**: `EDITOR`, `VISUAL`

## 🚀 **Quick Setup Guide**

### 1. **Minimal Setup**
```bash
# Copy examples
cp .env.example .env

# Edit .env with your API keys
GEMINI_API_KEY=your-gemini-key-here
GOOGLE_CUSTOM_SEARCH_API_KEY=your-google-key
GOOGLE_CUSTOM_SEARCH_ID=your-search-id
```

### 2. **Advanced Setup**
```bash
# Copy comprehensive config
cp .briefly.yaml.example .briefly.yaml

# Edit .briefly.yaml for detailed customization
# Set sensitive keys in .env file
```

### 3. **Verification**
```bash
# Test basic functionality
briefly digest input/test-links.md

# Test research with search
briefly research "AI trends" --depth 1
```

## 📚 **Configuration Methods**

### **Environment Variables** (Recommended for secrets)
```bash
# Primary names (recommended)
export GEMINI_API_KEY="your-key"
export GOOGLE_CUSTOM_SEARCH_API_KEY="your-key"
export GOOGLE_CUSTOM_SEARCH_ID="your-id"

# Alternative names (also supported)
export GOOGLE_GEMINI_API_KEY="your-key"      # Alternative Gemini
export GOOGLE_CSE_API_KEY="your-key"         # Alternative Google
export GOOGLE_CSE_ID="your-id"               # Alternative Google ID
```

### **YAML Configuration** (Recommended for settings)
```yaml
# .briefly.yaml
ai:
  gemini:
    model: "gemini-1.5-pro"    # Override default model
    temperature: 0.5           # More deterministic output

search:
  default_provider: "google"   # Use Google instead of DuckDuckGo
  max_results: 15             # More results per query

output:
  directory: "my-reports"      # Custom output directory
```

### **Command Line** (Recommended for one-time overrides)
```bash
# Override config for single command
briefly research "topic" --depth 2 --max-results 20
```

## 🔍 **Configuration Inspection**

### **View Current Configuration**
The centralized config module provides easy inspection:

```go
// In any Go file
import "briefly/internal/config"

// Get any configuration section
geminiConfig := config.GetAI().Gemini
searchConfig := config.GetSearch()
cacheConfig := config.GetCache()

// Get specific values
apiKey := config.GetGeminiAPIKey()
provider := config.GetSearchProvider()
outputDir := config.GetOutputDirectory()

// Check provider availability
if config.HasValidGoogleSearch() {
    // Google Custom Search is configured
}
```

### **Available Convenience Functions**
```go
// Section getters
config.GetApp()
config.GetAI()
config.GetSearch()
config.GetOutput()
config.GetCache()
config.GetVisual()
config.GetTTS()
config.GetMessaging()
config.GetEmail()
config.GetFeeds()
config.GetResearch()
config.GetLogging()
config.GetCLI()

// Specific value getters
config.GetGeminiAPIKey()
config.GetGeminiModel()
config.GetOpenAIAPIKey()
config.GetSearchProvider()
config.GetGoogleSearchConfig()  // Returns (apiKey, searchID)
config.GetSerpAPIKey()
config.GetOutputDirectory()
config.GetCacheDirectory()
config.IsDebugMode()

// Validation functions
config.HasValidGoogleSearch()
config.HasValidSerpAPI()
```

## 🛠️ **Adding New Configuration**

To add new configuration options:

1. **Add to Config Struct** in `/internal/config/config.go`
2. **Set Defaults** in `setDefaults()`
3. **Bind Environment Variables** in `bindEnvironmentVariables()`
4. **Add Validation** in `validateConfig()` if required
5. **Add Convenience Getter** for easy access
6. **Update Documentation** in this file

## 🔒 **Security Best Practices**

1. **Never commit API keys** to version control
2. **Use environment variables** for sensitive data
3. **Use YAML files** for non-sensitive settings
4. **Set appropriate file permissions**: `chmod 600 .env`
5. **Use different keys** for development/production
6. **Monitor API usage** for unexpected costs

## 🎯 **Benefits of Centralized Configuration**

### **For Developers**
- **Single Source of Truth**: All configuration in one place
- **Type Safety**: Strongly typed configuration structs
- **Easy Testing**: `config.Reset()` for clean test environments
- **Flexible Access**: Multiple ways to get the same value

### **For Users**
- **Consistent Interface**: Same patterns across all features
- **Flexible Setup**: Multiple naming conventions supported
- **Clear Documentation**: Complete overview in one place
- **Easy Troubleshooting**: Clear validation and error messages

### **For Deployment**
- **Environment Friendly**: Works well with containers and CI/CD
- **Override Friendly**: Easy to override settings per environment
- **Validation Built-in**: Catches configuration errors early
- **Backward Compatible**: Existing setups continue to work

This centralized configuration system makes Briefly much easier to configure, deploy, and maintain while providing a clear overview of all available options across the entire application.