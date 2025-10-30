# Interactive CLI Flow - Data Flow Specification

## Overview
This document specifies the data flow for the interactive CLI mode feature in Briefly, allowing users to interactively select "Game-Changer" articles and provide personal commentary.

## Primary Data Flow

```mermaid
graph TD
    Start[CLI Invocation] --> ModeCheck{Interactive Mode?}
    
    %% Standard Flow Branch
    ModeCheck -->|No| StandardFlow[Standard Automated Flow]
    StandardFlow --> AutoPriority[Auto-Calculate Priority Scores]
    AutoPriority --> AutoSelect[Auto-Select Game Changer]
    AutoSelect --> GenerateDigest[Generate Digest]
    
    %% Interactive Flow Branch
    ModeCheck -->|Yes| LoadArticles[Load & Process Articles]
    LoadArticles --> FetchContent[Fetch Article Content]
    FetchContent --> GenerateSummaries[Generate AI Summaries]
    GenerateSummaries --> CalculatePriority[Calculate Priority Scores]
    CalculatePriority --> SortArticles[Sort by Priority/Relevance]
    
    %% Interactive Selection Phase
    SortArticles --> PresentArticles[Present Article List]
    PresentArticles --> UserSelection{User Selects Article}
    UserSelection -->|Cancel| Abort[Exit Process]
    UserSelection -->|Select| CaptureSelection[Store Selected Article ID]
    
    %% Commentary Phase
    CaptureSelection --> PromptTake[Prompt for Personal Take]
    PromptTake --> UserInput{User Provides Take}
    UserInput -->|Skip| SkipTake[Mark Take as Empty]
    UserInput -->|Cancel| Abort
    UserInput -->|Input| ValidateTake{Validate Input}
    
    ValidateTake -->|Invalid| ShowError[Show Validation Error]
    ShowError --> PromptTake
    ValidateTake -->|Valid| StoreTake[Store Personal Take]
    
    %% Merge and Continue
    SkipTake --> MergeData[Merge Interactive Data]
    StoreTake --> MergeData
    MergeData --> UpdateArticles[Update Article Metadata]
    UpdateArticles --> GenerateDigest
    
    %% Final Output
    GenerateDigest --> RenderTemplate[Apply Template]
    RenderTemplate --> WriteOutput[Write Output File]
    WriteOutput --> End[Complete]
    Abort --> End
```

## Component Interaction Flow

```mermaid
sequenceDiagram
    participant CLI as CLI Controller
    participant IH as Interactive Handler
    participant AS as Article Service
    participant UI as UI Presenter
    participant VS as Validation Service
    participant DS as Data Store
    participant TG as Template Generator
    
    CLI->>CLI: Parse --interactive flag
    CLI->>IH: Initialize Interactive Mode
    
    IH->>AS: Request Processed Articles
    AS->>DS: Fetch Articles & Summaries
    DS-->>AS: Return Article Data
    AS->>AS: Calculate Priority Scores
    AS-->>IH: Return Sorted Articles
    
    IH->>UI: Present Article List
    UI-->>User: Display Selection Interface
    User-->>UI: Select Article (or Cancel)
    UI-->>IH: Return Selection
    
    alt User Cancels
        IH-->>CLI: Abort Process
    else User Selects Article
        IH->>UI: Prompt for Personal Take
        UI-->>User: Display Input Interface
        User-->>UI: Enter Commentary
        UI-->>IH: Return User Input
        
        IH->>VS: Validate Input
        VS-->>IH: Validation Result
        
        alt Invalid Input
            IH->>UI: Show Error & Retry
        else Valid Input
            IH->>DS: Store Selection & Take
            DS-->>IH: Confirm Storage
            
            IH->>TG: Generate with Interactive Data
            TG->>DS: Retrieve Enhanced Data
            DS-->>TG: Return Complete Dataset
            TG-->>CLI: Return Formatted Output
        end
    end
```

## Data Transformation Pipeline

```mermaid
graph LR
    subgraph Input Stage
        Raw[Raw Article Links] --> Parse[Parse Links]
        Parse --> Fetch[Fetch Content]
    end
    
    subgraph Processing Stage
        Fetch --> Extract[Extract Text]
        Extract --> Summarize[AI Summarization]
        Summarize --> Score[Priority Scoring]
    end
    
    subgraph Interactive Stage
        Score --> Present[Format for Display]
        Present --> Select[User Selection]
        Select --> Enhance[Add User Commentary]
    end
    
    subgraph Output Stage
        Enhance --> Merge[Merge with Template Data]
        Merge --> Render[Render Final Format]
        Render --> Output[Write Output]
    end
```

## Error Handling Paths

```mermaid
graph TD
    subgraph Error Scenarios
        E1[Invalid Selection] --> Retry1[Prompt Retry]
        E2[Empty Article List] --> Exit1[Graceful Exit]
        E3[Network Failure] --> Cache[Use Cached Data]
        E4[User Cancellation] --> Cleanup[Cleanup & Exit]
        E5[Template Error] --> Fallback[Use Default Template]
        E6[Storage Error] --> Memory[Use Memory Store]
    end
    
    Retry1 --> Recovery[Recovery Flow]
    Cache --> Recovery
    Fallback --> Recovery
    Memory --> Recovery
    
    Exit1 --> ErrorLog[Log Error]
    Cleanup --> ErrorLog
    ErrorLog --> Terminate[Terminate Process]
```

## State Transitions

```mermaid
stateDiagram-v2
    [*] --> Initializing
    Initializing --> LoadingArticles: Start Interactive Mode
    Initializing --> StandardMode: No Interactive Flag
    
    LoadingArticles --> ProcessingArticles: Articles Loaded
    LoadingArticles --> Error: Load Failed
    
    ProcessingArticles --> PresentingSelection: Articles Ready
    ProcessingArticles --> Error: Processing Failed
    
    PresentingSelection --> AwaitingSelection: UI Displayed
    AwaitingSelection --> ArticleSelected: User Selects
    AwaitingSelection --> Cancelled: User Cancels
    
    ArticleSelected --> PromptingForTake: Selection Stored
    PromptingForTake --> AwaitingInput: Prompt Displayed
    
    AwaitingInput --> ValidatingInput: Input Received
    AwaitingInput --> SkippedTake: User Skips
    AwaitingInput --> Cancelled: User Cancels
    
    ValidatingInput --> InputValid: Validation Passed
    ValidatingInput --> InputInvalid: Validation Failed
    
    InputInvalid --> PromptingForTake: Retry
    InputValid --> StoringData: Store Take
    SkippedTake --> StoringData: Store Empty Take
    
    StoringData --> GeneratingOutput: Data Persisted
    GeneratingOutput --> Complete: Output Generated
    
    StandardMode --> Complete: Standard Flow Complete
    Cancelled --> [*]
    Error --> [*]
    Complete --> [*]
```

## Async vs Sync Operations

| Operation | Type | Reason | Timeout |
|-----------|------|--------|---------|
| Article Fetching | Async (Parallel) | Multiple network requests | 30s per article |
| AI Summarization | Async (Batched) | API rate limits | 60s per batch |
| Priority Scoring | Sync | Fast calculation | N/A |
| UI Presentation | Sync | User interaction | N/A |
| User Input | Sync (Blocking) | Awaiting user | Configurable |
| Take Validation | Sync | Immediate feedback | N/A |
| Data Storage | Async | Non-blocking persistence | 5s |
| Template Rendering | Sync | Fast operation | N/A |
| File Writing | Async | I/O operation | 10s |

## Decision Points

1. **Mode Selection**: `--interactive` flag presence determines flow branch
2. **Article Selection**: User choice or timeout triggers next phase
3. **Take Input**: User can provide, skip, or cancel
4. **Validation**: Input must meet minimum length and format requirements
5. **Storage Strategy**: Memory-first with async persistence to disk
6. **Error Recovery**: Each failure point has defined recovery strategy
7. **Template Selection**: Based on format flag and interactive data presence