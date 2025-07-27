---
description: Use natural language to explore dataset results with DuckDB queries via gractl
argument-hint: <natural language query>
---

You are an expert data analyst helping explore datasets using natural language queries. 

For the query: "$ARGUMENTS"

Follow these steps:

1. **Parse the Query**: Understand the user's natural language request and identify what data they're looking for.

2. **Find Next Query Number**: Check the `/workspace/code` directory for existing SQL files to determine the next incremental number (e.g., 001, 002, 003).

3. **Generate SQL**: Create a DuckDB-compatible SQL query based on the natural language request. Consider:
   - Available tables and columns
   - Appropriate aggregations, filters, joins
   - Proper SQL syntax for DuckDB

4. **Create Intent-based Filename**: Generate a descriptive intent from the query (e.g., "total-sales-by-region", "user-signup-trends").

5. **Save and Execute**:
   - Save SQL as `{number}-sql-{intent}.sql` in `/workspace/code/`
   - Execute using: `gractl execute -f {sql_file} | tee /workspace/outputs/{number}-sql-{intent}.result.txt`

6. **Generate Report**: Create a markdown report with:
   - **Final Answer** (concise summary at the top)
   - **Query Intent** (what was being explored)
   - **SQL Query** (the generated SQL)
   - **Execution Steps** (brief explanation of approach)
   - **Key Findings** (important insights from results)

7. **File Management**: Ensure all files are properly saved with consistent naming for easy tracking and reproduction.

## Example

For query: "Show me the top 5 customers by total order value"

- Intent: `top-customers-by-order-value`
- SQL file: `001-sql-top-customers-by-order-value.sql`
- Result file: `001-sql-top-customers-by-order-value.result.txt`
- Generate comprehensive markdown report with findings

Always start with finding the next available number and create all necessary directories if they don't exist.
