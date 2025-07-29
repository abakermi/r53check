# r53check

A simple Go command-line tool to check domain availability using AWS Route 53 Domains API.

## Prerequisites

- Go 1.19 or newer
- AWS credentials configured (via AWS CLI, environment variables, or IAM roles)
- AWS account with Route 53 Domains API access

## Installation

Install directly from GitHub:

```sh
go install github.com/abakermi/r53check/cmd@latest
```

Or clone the repository and build locally:

```sh
git clone https://github.com/abakermi/r53check.git
cd r53check
go build -o r53check ./cmd/
```

## AWS Configuration

The tool requires AWS credentials to access the Route 53 Domains API. You can configure them using:

### AWS CLI (Recommended)

```sh
aws configure
```

### Environment Variables

```sh
export AWS_ACCESS_KEY_ID=your_access_key
export AWS_SECRET_ACCESS_KEY=your_secret_key
export AWS_REGION=us-east-1
```

### IAM Permissions

Your AWS credentials need the following permission:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": ["route53domains:CheckDomainAvailability"],
      "Resource": "*"
    }
  ]
}
```

## Usage

### Single Domain Check

Check if a domain is available for registration:

```sh
r53check check <domain_name>
```

Examples:

```sh
# Check a .com domain
r53check check example.com

# Check with verbose output
r53check --verbose check myapp.io

# Check with custom timeout
r53check --timeout 30s check example.org
```

### Bulk Domain Check

Check multiple domains at once:

```sh
r53check bulk <domain1> <domain2> <domain3>...
```

Or read domains from a file:

```sh
r53check bulk --file domains.txt
```

Examples:

```sh
# Check multiple domains
r53check bulk example.com test.org myapp.io

# Check domains from file with verbose output
r53check --verbose bulk --file domains.txt

# Check with custom timeout
r53check --timeout 30s bulk example.com test.com
```

#### Domains File Format

Create a text file with one domain per line:

```
example.com
test.org
myapp.io
# This is a comment and will be ignored
another-domain.net
```

### Global Flags

- `--timeout duration`: Set timeout for API requests (default: 10s)
- `--region string`: AWS region (defaults to us-east-1)
- `--verbose, -v`: Enable verbose output

## Output

The tool provides clear output indicating domain availability:

- ✓ **AVAILABLE**: Domain is available for registration
- ✗ **UNAVAILABLE**: Domain is already registered
- ⚠ **RESERVED**: Domain is reserved and cannot be registered
- ? **UNKNOWN**: Unable to determine availability

## Exit Codes

- `0`: Success (domain checked successfully)
- `1`: Validation error (invalid domain format)
- `2`: Authentication error (AWS credentials issue)
- `3`: Authorization error (insufficient permissions)
- `4`: API error (AWS service error)
- `5`: System error (unexpected error)

## Supported TLDs

The tool supports checking availability for common TLDs including:

- Generic: .com, .net, .org, .info, .biz, .name, .io, .co, .me, .tv, .cc, .ws, .mobi, .tel, .asia
- Country codes: .us, .uk, .ca, .au, .de, .fr, .it, .es, .nl, .be, .ch, .at, .se, .no, .dk, .fi, .pl, .cz, .ru, .jp, .cn, .in, .br, .mx

## Help

You can see usage instructions with:

```sh
r53check --help
r53check check --help
r53check bulk --help
```

## Development

### Building

```sh
go build -o r53check ./cmd/
```

### Testing

```sh
go test ./...
```

### Project Structure

```
r53check/
├── cmd/                    # Main application entry point
├── internal/
│   ├── aws/               # AWS Route 53 client wrapper
│   ├── domain/            # Domain validation and checking logic
│   ├── errors/            # Custom error types and handling
│   └── output/            # Output formatting
├── go.mod
├── go.sum
└── README.md
```

## License

MIT
