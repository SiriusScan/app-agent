# Example Templates

Real-world example templates demonstrating all features.

## Purpose

- Provide working examples for template authors
- Demonstrate best practices
- Show all module capabilities
- Serve as reference documentation

## Examples

- `openssh-vulnerable.yaml` - Detect vulnerable OpenSSH versions
- `apache-misconfiguration.yaml` - Detect Apache misconfigurations
- `nginx-version-detection.yaml` - Detect Nginx versions
- `multi-step-detection.yaml` - Complex multi-step detection

## Usage

```bash
# Run a single example
./agent template run templates/examples/openssh-vulnerable.yaml

# Run all examples
./agent template run-all templates/examples/
```

