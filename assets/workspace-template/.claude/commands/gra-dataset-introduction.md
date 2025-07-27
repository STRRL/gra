---
description: "Explore dataset in remote runner using gractl and DuckDB, then generate comprehensive dataset introduction"
tools: ["Bash", "Write", "Edit"]
---

# Dataset Insight Analysis

You are tasked with exploring a dataset located at `/workspace/dataset` on a remote runner using gractl. Follow these steps systematically:

## Step 1: Initial Dataset Discovery

Use gractl to explore the dataset structure:

- Check if `/workspace/dataset` exists and is accessible
- **IMPORTANT**: If `/workspace/dataset` is empty or doesn't exist, simply report this fact and stop analysis. Do NOT create example data or hallucinate dataset contents.
- Focus discovery on structured data files: CSV, JSON, NDJSON, and Parquet files only
- Use targeted find commands to avoid listing too many small files that could crash ls
- Check file sizes and modification dates for discovered data files only
- Examine directory structure for data organization patterns

## Step 2: Dataset Schema Analysis

For each data file discovered:

- Use DuckDB via gractl to analyze file schemas
- Identify column names, types, and constraints
- Count total records in each file
- Check for missing values or data quality issues

## Step 3: Statistical Analysis

Perform comprehensive statistical analysis using DuckDB:

- Generate descriptive statistics for numerical columns
- Analyze categorical distributions
- Identify potential data quality issues
- Calculate key metrics and percentages
- Look for demographic distributions, class imbalances, or other patterns

## Step 4: Data Relationships

If multiple files exist:

- Identify potential relationships between files
- Check for foreign key relationships
- Analyze data consistency across files

## Step 5: Generate Dataset Introduction

Create a comprehensive `dataset-introduction.md` file containing:

- **Overview**: Dataset name, purpose, and source
- **Structure**: File organization and format details
- **Schema**: Detailed column descriptions and data types
- **Statistics**: Key metrics, distributions, and insights
- **Data Quality**: Any issues or considerations found
- **Use Cases**: Potential applications and research directions
- **Technical Notes**: Access patterns, performance considerations

## Execution Commands

Use these gractl command patterns:

```bash
# Safe dataset discovery - focus on structured data files only
gractl execute "ls -ld /workspace/dataset"
gractl execute "find /workspace/dataset -type f \( -name '*.csv' -o -name '*.json' -o -name '*.ndjson' -o -name '*.parquet' \) | head -20"
gractl execute "find /workspace/dataset -type f \( -name '*.csv' -o -name '*.json' -o -name '*.ndjson' -o -name '*.parquet' \) -exec ls -lh {} \; | head -20"

# DuckDB analysis
gractl execute "duckdb -c \"DESCRIBE SELECT * FROM '/workspace/dataset/path/file.csv';\""
gractl execute "duckdb -c \"SELECT COUNT(*) FROM '/workspace/dataset/path/file.csv';\""
gractl execute "duckdb -c \"SELECT column_name, COUNT(*) FROM '/workspace/dataset/path/file.csv' GROUP BY column_name ORDER BY COUNT(*) DESC LIMIT 10;\""

# Statistical analysis
gractl execute "duckdb -c \"SELECT AVG(column), MIN(column), MAX(column), STDDEV(column) FROM '/workspace/dataset/path/file.csv';\""
```

## Output Requirements

- **If no dataset found**: Simply report "No dataset found in /workspace/dataset" and stop
- **If dataset exists**: Be thorough in your analysis
- Include specific numbers and percentages
- Highlight any interesting patterns or anomalies
- Provide actionable insights for researchers/developers
- Create a well-structured markdown document with clear sections
- **Never create fake or example data**

Arguments: $ARGUMENTS (optional: specify subdirectory to focus analysis)
