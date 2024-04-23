# tfc-rum-stats

Utility script to generate Terraform Cloud RUM stats.

At the end a CSV is generated listing each workspace and its resource count.

## Usage

List RUM stats for all organizations you have access to:

```bash
TFE_TOKEN=<tfe token here> go run ./cmd/tfc-rum-stats/main.go -all
```

List rUM stats for a specific organization:

```bash
TFE_TOKEN=<tfe token here> go run ./cmd/tfc-rum-stats/main.go -name <org name>
```

Add `-verbose` to either command to get more output or `-h` to view instructions.
