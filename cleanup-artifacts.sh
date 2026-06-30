#!/bin/bash
# Script to delete old GitHub Actions workflow runs and their artifacts

echo "Fetching workflow runs..."

# Get all workflow run IDs older than 7 days
run_ids=$(gh run list --repo nowafinance/nowa-zk --limit 100 --json databaseId,createdAt,conclusion --jq '.[] | select(.createdAt < (now - 604800 | todate)) | .databaseId')

if [ -z "$run_ids" ]; then
    echo "No old runs found to delete."
    exit 0
fi

echo "Found $(echo "$run_ids" | wc -l) old runs to delete."
echo "Deleting runs..."

for run_id in $run_ids; do
    echo "Deleting run $run_id..."
    gh run delete "$run_id" --repo nowafinance/nowa-zk
done

echo "Cleanup complete!"
