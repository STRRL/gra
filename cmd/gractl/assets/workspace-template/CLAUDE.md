# CLAUDE.md - gractl Workspace Setup Guide

This file provides guidance for setting up and using Claude Code with gractl for remote code execution.

## Quick Start

1. **Initialize your workspace**: Run `gractl workspace init` to create this template
2. **Configure gractl**: Copy `.gractl.example.toml` to `.gractl.toml` and customize
3. **Use Claude commands**: Execute `/execute` or `/analyze` to work with remote runners

## gractl Configuration Setup

### Step 1: Create .gractl.toml

Copy the example configuration and customize for your environment:

```bash
cp .gractl.example.toml .gractl.toml
```

### Step 2: Configure S3 Access

Edit `.gractl.toml` with your S3 credentials:

```toml
[server]
address = "localhost:9090"  # gractl server address

[s3]
bucket = "your-data-bucket"          # S3 bucket containing your datasets
endpoint = ""                        # Leave empty for AWS S3
prefix = "projects/my-project"       # Optional: S3 path prefix  
region = "us-east-1"                 # AWS region
access_key_id = "AKIA..."           # Your AWS access key
secret_access_key = "xxxx..."        # Your AWS secret key
read_only = false                    # Set to true for read-only access
```

## Available Claude Commands

This workspace includes two specialized slash commands in `.claude/commands/` for working with gractl:

### /gra-dataset-introduction - Dataset Exploration

Automatically explore and analyze datasets in your remote runner, generating comprehensive documentation.

**Usage**: `/gra-dataset-introduction [optional-subdirectory]`

**What it does**:

- Explores dataset structure at `/workspace/dataset`
- Uses DuckDB to analyze file schemas and statistics
- Generates comprehensive `dataset-introduction.md` with insights
- Focuses on CSV, JSON, NDJSON, and Parquet files
- Provides data quality assessment and use case recommendations

### /gra-query - Natural Language Data Queries

Convert natural language questions into DuckDB SQL queries and execute them remotely.

**Usage**: `/gra-query Show me the top 5 customers by total order value`

**What it does**:

- Parses your natural language query into SQL
- Creates numbered SQL files in `/workspace/code/`
- Executes queries using gractl with results saved to `/workspace/outputs/`
- Generates markdown reports with findings and insights
- Maintains query history for reproducible analysis

## Working with Remote Runners

### Basic Workflow

1. **Execute commands**: Use `gractl execute "your-command"` or the `/execute` Claude command
2. **Data access**: Your S3 data is automatically mounted at `/workspace/dataset`
3. **File sync**: Use `gractl workspace sync` to mount remote files locally

### Example Commands

```bash
# Quick analysis with auto-runner creation
gractl execute "ls -la /workspace/dataset"

# Create a specific runner with S3 workspace
gractl runners create --name analysis-runner --s3-bucket my-data

# Execute commands in a specific runner
gractl runners exec analysis-runner "python script.py"

# Sync workspace locally for file editing
gractl workspace sync analysis-runner
```

## Environment Structure

```
your-workspace/
├── CLAUDE.md                        # This guide
├── .gractl.toml                     # gractl configuration
├── .gractl.example.toml             # Configuration template
├── .claude/
│   └── commands/
│       ├── gra-dataset-introduction.md  # Dataset exploration command
│       └── gra-query.md                 # Natural language SQL queries
└── runners/                         # Local workspace sync directory (created by workspace sync)
    └── runner-*/
        └── workspace/               # Synced remote runner files
```

## Data Access

### S3 Data Location

- **Remote mount point**: `/workspace/dataset` (hardcoded)
- **Contains**: All files from your configured S3 bucket/prefix
- **Access**: Direct file operations, no special S3 APIs needed

### Local File Sync

- **Command**: `gractl workspace sync runner-name`
- **Local path**: `./runners/runner-name/workspace/`
- **Method**: SSH + sshfs for secure file synchronization

## Best Practices

### Security

- Add `.gractl.toml` to your `.gitignore` if it contains credentials
- Use read-only S3 access when possible

### Performance

- Use specific S3 prefixes to limit data scope
- Clean up unused runners with `gractl runners delete runner-name`
- Use `--timeout` for long-running operations

### Development Workflow

1. Use `/gra-dataset-introduction` to explore and document your data
2. Use `/gra-query` to ask natural language questions about your data
3. Use `workspace sync` to edit files locally when needed
4. Iterate between remote execution and local development

## Example Session

```bash
# Start by exploring your dataset
# /gra-dataset-introduction

# Ask questions about your data
# /gra-query What are the sales trends by month?

# Execute commands directly if needed
gractl execute "ls /workspace/dataset"

# Sync files locally for editing
gractl workspace sync

# Execute your analysis remotely  
gractl execute "python analysis.py"
```

This setup provides a seamless bridge between local Claude Code interaction and remote data processing capabilities via gractl.
