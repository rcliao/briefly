# Interactive CLI Flow - Verification Criteria

## Component Verification Checklist

### 1. CLI Controller Verification

#### Functional Verification
- [ ] Accepts `--interactive` flag
- [ ] Validates source file exists and is readable
- [ ] Parses all command-line arguments correctly
- [ ] Routes to interactive handler when flag present
- [ ] Maintains backward compatibility with existing flags
- [ ] Handles invalid flag combinations gracefully

#### Error Handling Verification
- [ ] Returns appropriate exit codes (0=success, 1=error, 2=cancelled)
- [ ] Displays user-friendly error messages
- [ ] Cleans up resources on abnormal termination
- [ ] Logs errors with sufficient detail for debugging

#### Test Commands
```bash
# Verify interactive mode activation
briefly digest --interactive input/test.md

# Verify standard mode still works
briefly digest input/test.md

# Verify error handling
briefly digest --interactive nonexistent.md
```

---

### 2. Interactive Handler Verification

#### State Management
- [ ] Creates new session with unique ID
- [ ] Maintains session state throughout flow
- [ ] Handles state transitions correctly
- [ ] Expires sessions after timeout
- [ ] Recovers from interrupted sessions

#### Flow Control
- [ ] Enforces correct sequence: Load -> Present -> Select -> Take -> Generate
- [ ] Prevents invalid state transitions
- [ ] Handles user cancellation at any point
- [ ] Supports skip options where appropriate

#### Verification Steps
1. Start interactive session and verify session ID creation
2. Navigate through each state and verify transitions
3. Test timeout by leaving session idle
4. Test cancellation at each stage
5. Verify session cleanup after completion

---

### 3. UI Presenter Verification

#### Display Verification
- [ ] Shows article list with correct formatting
- [ ] Displays priority scores and indicators
- [ ] Updates selection highlight on navigation
- [ ] Shows current position (e.g., "3 of 15")
- [ ] Handles terminal resize gracefully
- [ ] Supports both color and no-color modes

#### Navigation Verification
- [ ] Arrow keys move selection up/down
- [ ] Page Up/Down work correctly
- [ ] Home/End jump to first/last
- [ ] Number input jumps to article
- [ ] Enter selects current article
- [ ] ESC cancels operation

#### Input Verification
- [ ] Text input area appears for user take
- [ ] Supports multi-line input
- [ ] Shows character/word count
- [ ] Handles special characters
- [ ] Supports basic editing (backspace, delete)

#### Test Scenarios
```yaml
navigation_test:
  - Press Down 5 times -> Verify position 6
  - Press Page Down -> Verify next page
  - Press Home -> Verify position 1
  - Type "5" -> Verify jump to article 5
  - Press Enter -> Verify selection confirmed

display_test:
  - Load 20 articles -> Verify pagination
  - Resize terminal -> Verify reflow
  - Use NO_COLOR=1 -> Verify plain text mode
```

---

### 4. Article Service Verification

#### Processing Pipeline
- [ ] Fetches article content successfully
- [ ] Extracts main text from HTML
- [ ] Generates AI summaries
- [ ] Handles fetch failures gracefully
- [ ] Uses cache when available
- [ ] Processes articles in parallel

#### Cache Behavior
- [ ] Stores fetched content for 24 hours
- [ ] Stores summaries for 7 days
- [ ] Validates cache with content hash
- [ ] Falls back to cache on network failure
- [ ] Respects force-refresh flag

#### Performance Criteria
| Metric | Target | Acceptable |
|--------|--------|------------|
| Single article fetch | < 2s | < 5s |
| Batch of 10 articles | < 10s | < 20s |
| Cache hit rate | > 80% | > 60% |
| Summary generation | < 3s | < 6s |

---

### 5. Priority Service Verification

#### Scoring Algorithm
- [ ] Calculates scores based on configured weights
- [ ] Considers recency, relevance, sentiment
- [ ] Produces scores in range [0.0, 1.0]
- [ ] Sorts articles by score descending
- [ ] Identifies top article as potential game-changer

#### Verification Data
```yaml
test_articles:
  - title: "Breaking: Major Tech Announcement"
    age: 1_hour
    sentiment: positive
    expected_score: > 0.8
    
  - title: "Weekly Roundup"
    age: 3_days
    sentiment: neutral
    expected_score: 0.4 - 0.6
    
  - title: "Deprecated Library Notice"
    age: 1_week
    sentiment: negative
    expected_score: < 0.3
```

---

### 6. Take Handler Verification

#### Input Handling
- [ ] Captures multi-line text correctly
- [ ] Preserves formatting (line breaks, spaces)
- [ ] Handles paste operations
- [ ] Supports cancel without saving
- [ ] Allows skipping with confirmation

#### Validation
- [ ] Enforces minimum length (10 characters)
- [ ] Enforces maximum length (2000 characters)
- [ ] Provides clear validation messages
- [ ] Allows retry on validation failure
- [ ] Accepts empty take when explicitly skipped

#### Storage
- [ ] Associates take with correct article
- [ ] Persists take to session store
- [ ] Includes take in final digest
- [ ] Preserves take through digest generation

---

### 7. Session Store Verification

#### Session Management
- [ ] Creates sessions atomically
- [ ] Updates session state transactionally
- [ ] Prevents concurrent modification
- [ ] Expires sessions after timeout
- [ ] Cleans up expired sessions

#### Data Integrity
- [ ] Session data persists across operations
- [ ] Selections are not lost
- [ ] User takes are preserved
- [ ] Recovery possible after crash

#### Verification Queries
```sql
-- Verify session creation
SELECT * FROM sessions WHERE session_id = ?;

-- Check active sessions
SELECT COUNT(*) FROM sessions WHERE expires_at > NOW();

-- Verify cleanup
SELECT COUNT(*) FROM sessions WHERE expires_at < NOW() - INTERVAL '1 hour';
```

---

### 8. Template Generator Verification

#### Template Rendering
- [ ] Includes selected game-changer at top
- [ ] Shows user take with article
- [ ] Maintains format consistency
- [ ] Handles missing optional data
- [ ] Produces valid markdown/HTML

#### Format Support
- [ ] Newsletter format includes all elements
- [ ] Email format is HTML-compatible
- [ ] Standard format is readable
- [ ] Brief format is concise

#### Verification Output
```markdown
# Expected Structure

## 🎯 Game-Changer: [Selected Article Title]

[Article Summary]

**My Take:** [User Commentary]

---

## Other Articles
[Remaining articles in priority order]
```

---

## Integration Test Scenarios

### Scenario 1: Happy Path
```yaml
test: "Complete Interactive Flow"
steps:
  1. Start with --interactive flag
  2. Wait for articles to load
  3. Navigate to article 3
  4. Select with Enter
  5. Input personal take
  6. Confirm and generate
expected:
  - Digest generated with selected article as game-changer
  - User take included in output
  - Other articles listed below
```

### Scenario 2: Cancellation Flow
```yaml
test: "User Cancellation"
steps:
  1. Start interactive mode
  2. Navigate to article
  3. Press ESC to cancel
expected:
  - Process exits cleanly
  - No output file created
  - Exit code = 2
```

### Scenario 3: Validation Failure
```yaml
test: "Take Validation"
steps:
  1. Select article
  2. Enter take with only 5 characters
  3. Observe validation error
  4. Enter valid take
expected:
  - First submission rejected
  - Clear error message shown
  - Second submission accepted
```

### Scenario 4: Timeout Handling
```yaml
test: "Session Timeout"
steps:
  1. Start interactive mode
  2. Leave idle for timeout period
expected:
  - Session expires
  - User notified
  - Graceful exit
```

---

## Performance Verification

### Response Time Requirements
| Action | Target | Maximum |
|--------|--------|---------|
| Navigation key press | < 50ms | 100ms |
| Article list display | < 500ms | 1s |
| Article selection | < 100ms | 200ms |
| Take submission | < 200ms | 500ms |
| Digest generation | < 5s | 10s |

### Load Testing
```yaml
load_test:
  articles: 50
  expected:
    - Initial load: < 15s
    - Navigation: responsive
    - Memory usage: < 200MB
    - CPU usage: < 50%
```

---

## Acceptance Criteria

### User Experience
- [ ] User can easily navigate article list
- [ ] Selection process is intuitive
- [ ] Take input feels natural
- [ ] Feedback is immediate and clear
- [ ] Cancellation is always available

### Data Integrity
- [ ] Selected article appears as game-changer
- [ ] User take is included verbatim
- [ ] Article order reflects selection
- [ ] No data loss during process

### Backward Compatibility
- [ ] Standard mode works unchanged
- [ ] Existing flags still function
- [ ] Output format unchanged for standard mode
- [ ] Cache format compatible

### Error Recovery
- [ ] Network failures handled gracefully
- [ ] AI service failures don't break flow
- [ ] Invalid input doesn't crash
- [ ] Partial failures are isolated

---

## Monitoring and Observability

### Metrics to Track
```yaml
metrics:
  usage:
    - interactive_sessions_started
    - interactive_sessions_completed
    - interactive_sessions_cancelled
    
  performance:
    - article_fetch_time_p50
    - article_fetch_time_p99
    - ui_response_time_p99
    
  errors:
    - validation_failures
    - timeout_occurrences
    - api_failures
    
  user_behavior:
    - average_navigation_actions
    - take_skip_rate
    - average_take_length
```

### Logging Requirements
```yaml
log_levels:
  INFO:
    - Session started/completed
    - Article selected
    - Take submitted
    
  WARN:
    - Validation failures
    - Timeout warnings
    - Cache misses
    
  ERROR:
    - API failures
    - Unhandled exceptions
    - Data corruption
```

---

## Definition of Done

A component is considered complete when:

1. **All verification criteria are met** ✓
2. **Unit tests pass with >80% coverage** ✓
3. **Integration tests pass** ✓
4. **Performance targets achieved** ✓
5. **Error scenarios handled** ✓
6. **Documentation updated** ✓
7. **Code reviewed and approved** ✓
8. **Manual testing completed** ✓
9. **Backward compatibility verified** ✓
10. **Monitoring in place** ✓