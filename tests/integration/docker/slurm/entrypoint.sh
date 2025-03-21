#!/bin/bash
# Entrypoint script for Slurm container

set -e

export PATH=/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin

mkdir -p /run/dbus
chmod 755 /run/dbus
dbus-daemon --system --fork

# Start munge authentication service
echo "Starting munge daemon..."
# Run munged in background with default options
gosu munge /usr/sbin/munged

# Check munge is working
echo "Testing munge authentication..."
munge -n | unmunge || {
    echo "Munge authentication test failed"
    exit 1
}
echo "Munge authentication working"

# Print Slurm configuration information for debugging
echo "Slurm Configuration:"
slurmd -C || echo "WARNING: slurmd configuration check failed"

# Determine what Slurm component to run based on the first argument
case "$1" in
  slurmctld)
    echo "Starting slurmctld..."
    shift
    exec slurmctld "$@"
    ;;
  slurmd)
    echo "Starting slurmd..."
    shift
    # Use --conf-server option to get configuration from slurmctld
    exec slurmd "$@"
    ;;
  slurmrestd)
    echo "Starting slurmrestd..."
    shift
    # REST API mode with basic auth
    exec slurmrestd "$@"
    ;;
  scontrol|sinfo|sacct|squeue|sbatch|srun)
    # Slurm client commands
    exec "$@"
    ;;
  bash|sh)
    # Shell access
    exec "$@"
    ;;
  --help|help|-h)
    echo "Usage: <container> COMMAND [ARGS...]"
    echo ""
    echo "Commands:"
    echo "  slurmctld [args]    - Start Slurm controller daemon"
    echo "  slurmd [args]       - Start Slurm compute node daemon"
    echo "  slurmrestd [args]   - Start Slurm REST API daemon"
    echo "  scontrol [args]     - Run scontrol command"
    echo "  sinfo [args]        - Run sinfo command"
    echo "  sacct [args]        - Run sacct command"
    echo "  squeue [args]       - Run squeue command"
    echo "  sbatch [args]       - Run sbatch command"
    echo "  srun [args]         - Run srun command"
    echo "  bash|sh             - Start a shell"
    echo ""
    exit 0
    ;;
  *)
    echo "Unknown command: $1"
    echo "Run with --help for usage information"
    exit 1
    ;;
esac
