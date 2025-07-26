# S3 Workspace Mounting Examples

This document demonstrates how to mount S3 buckets as workspaces in runners using the gra system.

## Basic S3 Mounting

Create a runner with S3 bucket mounted as workspace:

```bash
# Create a runner with S3 workspace
gractl runners create \
  --name my-data-runner \
  --s3-bucket my-data-bucket \
  --s3-region us-west-2 \
  -e AWS_ACCESS_KEY_ID=your-access-key \
  -e AWS_SECRET_ACCESS_KEY=your-secret-key
```

## Advanced S3 Configuration

Create a runner with custom S3 configuration:

```bash
# Create a runner with custom S3 configuration
gractl runners create \
  --name analytics-runner \
  --s3-bucket analytics-data \
  --s3-prefix datasets/2024/ \
  --s3-region us-east-1 \
  --mount-path /data \
  --read-only \
  -e AWS_ACCESS_KEY_ID=your-access-key \
  -e AWS_SECRET_ACCESS_KEY=your-secret-key
```

## S3-Compatible Services (MinIO)

Create a runner with MinIO or other S3-compatible service:

```bash
# Create a runner with MinIO
gractl runners create \
  --name minio-runner \
  --s3-bucket test-bucket \
  --s3-endpoint http://minio.example.com:9000 \
  --mount-path /workspace \
  -e AWS_ACCESS_KEY_ID=minioaccess \
  -e AWS_SECRET_ACCESS_KEY=miniosecret
```

## Auto-Creation with Execute

The execute command automatically creates runners if none are available:

```bash
# Execute a command - will auto-create runner if needed
gractl execute "ls -la /workspace"
```

Note: The execute command currently doesn't accept workspace configuration flags. 
If you need S3 mounting with auto-created runners, first create a runner with 
the desired S3 configuration.

## Environment Variables

When mounting S3 buckets, you need to provide AWS credentials via environment variables:

- `AWS_ACCESS_KEY_ID`: Your AWS access key ID
- `AWS_SECRET_ACCESS_KEY`: Your AWS secret access key
- `AWS_SESSION_TOKEN`: (Optional) Session token for temporary credentials

## S3FS Configuration

The S3FS sidecar container receives the following environment variables:

- `S3_BUCKET`: S3 bucket name
- `S3_ENDPOINT`: S3 endpoint URL (optional)
- `S3_PREFIX`: Path prefix within bucket (optional)
- `AWS_DEFAULT_REGION`: AWS region (optional)
- `MOUNT_PATH`: Container mount path
- `MOUNT_OPTIONS`: Mount options (e.g., "ro" for read-only)

## Workspace Configuration Fields

| Field | Description | Default |
|-------|-------------|---------|
| `bucket` | S3 bucket name | Required |
| `endpoint` | S3 endpoint URL | AWS S3 |
| `prefix` | Path prefix in bucket | None |
| `region` | AWS region | us-east-1 |
| `mount_path` | Container mount path | /workspace |
| `read_only` | Read-only mount | false |

## Use Cases

### Data Analytics
```bash
# Create a runner for data analysis with read-only access to data lake
gractl runners create \
  --name data-analyst \
  --s3-bucket company-data-lake \
  --s3-prefix analytics/input/ \
  --read-only \
  -e AWS_ACCESS_KEY_ID=$AWS_ACCESS_KEY_ID \
  -e AWS_SECRET_ACCESS_KEY=$AWS_SECRET_ACCESS_KEY
```

### Machine Learning
```bash
# Create a runner for ML training with model artifacts
gractl runners create \
  --name ml-trainer \
  --s3-bucket ml-models \
  --s3-prefix experiments/exp-001/ \
  --mount-path /models \
  -e AWS_ACCESS_KEY_ID=$AWS_ACCESS_KEY_ID \
  -e AWS_SECRET_ACCESS_KEY=$AWS_SECRET_ACCESS_KEY
```

### Content Processing
```bash
# Create a runner for processing uploaded content
gractl runners create \
  --name content-processor \
  --s3-bucket user-uploads \
  --s3-prefix pending-processing/ \
  --mount-path /input \
  -e AWS_ACCESS_KEY_ID=$AWS_ACCESS_KEY_ID \
  -e AWS_SECRET_ACCESS_KEY=$AWS_SECRET_ACCESS_KEY
```

## Troubleshooting

1. **Mount fails**: Check AWS credentials and bucket permissions
2. **Empty workspace**: Verify bucket name and prefix are correct
3. **Access denied**: Ensure IAM user has s3:GetObject permissions
4. **Connection timeout**: Check S3 endpoint URL and network connectivity

## Security Considerations

- Use IAM roles when running in AWS instead of access keys
- Apply principle of least privilege to S3 bucket permissions
- Use read-only mounts when possible
- Consider bucket policies for additional security
- Never commit AWS credentials to code repositories