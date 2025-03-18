# Troubleshooting

This document describes potential issues and solutions when using prometheus-slurm-sd.

## Common Problems and Solutions

### Server Doesn't Start

#### 1. Configuration File Not Found

**Symptom**: You see an error message: `Failed to load config: failed to open config file: open config.yaml: no such file or directory`.

**Solution**:
- Ensure the configuration file exists in the specified location.
- Specify the correct path with the `--config.file` option.

```bash
./prometheus-slurm-sd --config.file=/path/to/your/config.yaml
```

#### 2. Slurm API Endpoint Not Specified

**Symptom**: You see an error message: `Slurm API endpoint is required`.

**Solution**:
- Set the `slurm_api_endpoint` in the configuration file or specify it with the command-line option `--slurm.api-endpoint`.

```yaml
# config.yaml
slurm_api_endpoint: "http://slurm-restd:6820"
```

Or

```bash
./prometheus-slurm-sd --slurm.api-endpoint="http://slurm-restd:6820"
```

#### 3. Address Already in Use

**Symptom**: You see an error message: `HTTP server error: listen tcp :8080: bind: address already in use`.

**Solution**:
- Change the port in the configuration to use a different one.

```yaml
# config.yaml
listen_address: ":8081"
```

Or

```bash
./prometheus-slurm-sd --web.listen-address=":8081"
```

### Cannot Connect to Slurm API

#### 1. Network Connection Issues

**Symptom**: You see an error message: `Failed to get nodes from Slurm: failed to execute request: dial tcp: lookup slurm-restd: no such host`.

**Solution**:
- Verify that the Slurm API endpoint hostname can be resolved.
- Try using an IP address instead of a hostname.
- Check your network connection.

#### 2. Authentication Error

**Symptom**: You see an error message: `Unexpected status code: 401, body: {"error":"Invalid Authentication","status":401}`.

**Solution**:
- Verify that JWT authentication credentials are correct.
- Specify username and token in the configuration file or with command-line options.

```yaml
# config.yaml
slurm_api_username: "username"
slurm_api_token: "your-token"
```

#### 3. API Version Mismatch

**Symptom**: You see an error message: `Unexpected status code: 404, body: {"error":"Not Found","status":404}`.

**Solution**:
- Check the Slurm REST API version and set the correct version in your configuration.

```yaml
# config.yaml
slurm_api_version: "v0.0.38"  # Change to match your actual version
```

### Targets Not Appearing in Prometheus

#### 1. Empty Response

**Symptom**: No service discovery targets appear in the Prometheus `/targets` page.

**Solution**:
- Check the logs of prometheus-slurm-sd to see if it's successfully retrieving node information from Slurm.
- Test the API with curl command.

```bash
curl http://localhost:8080/targets?prom_job=node
```

- If the returned JSON is empty, there might be no active nodes in the Slurm cluster, or the node information retrieval might have failed.

#### 2. Incorrect Prometheus Configuration

**Symptom**: prometheus-slurm-sd seems to be working correctly, but targets don't appear in Prometheus.

**Solution**:
- Verify that your Prometheus configuration file (prometheus.yml) is correctly configured.
- Ensure the URL is correct.
- Check the network connection between Prometheus and prometheus-slurm-sd.

```yaml
scrape_configs:
  - job_name: 'slurm-node-exporter'
    http_sd_configs:
      - url: http://prometheus-slurm-sd:8080/targets?prom_job=node
        refresh_interval: 5m
```

## Changing Log Level

To get more detailed debug information, set the log level to `debug`.

```bash
./prometheus-slurm-sd --log.level=debug
```

This will show details about Slurm API requests, target updates, and Prometheus request information.

## Support and Feedback

If you need further assistance or encounter new issues, please report them on the GitHub Issues page:

[https://github.com/yuuki/prometheus-slurm-sd/issues](https://github.com/yuuki/prometheus-slurm-sd/issues)
