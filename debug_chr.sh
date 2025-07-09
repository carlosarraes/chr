set -e

echo "=== Chr Debug Script ==="
echo "Current directory: $(pwd)"
echo "Current branch: $(git branch --show-current)"

echo ""
echo "=== Git Remote Info ==="
git remote -v || echo "No remotes found"

echo ""
echo "=== Branch Check ==="
echo "ZUP-35-prd exists: $(git rev-parse --verify ZUP-35-prd >/dev/null 2>&1 && echo 'YES' || echo 'NO')"
echo "ZUP-35-hml exists: $(git rev-parse --verify ZUP-35-hml >/dev/null 2>&1 && echo 'YES' || echo 'NO')"
echo "origin/ZUP-35-prd exists: $(git rev-parse --verify origin/ZUP-35-prd >/dev/null 2>&1 && echo 'YES' || echo 'NO')"
echo "origin/ZUP-35-hml exists: $(git rev-parse --verify origin/ZUP-35-hml >/dev/null 2>&1 && echo 'YES' || echo 'NO')"

echo ""
echo "=== Recent Commits in ZUP-35-prd ==="
git log --oneline -n 5 ZUP-35-prd 2>/dev/null || echo "Failed to get commits from ZUP-35-prd"

echo ""
echo "=== Recent Commits in origin/ZUP-35-prd ==="
git log --oneline -n 5 origin/ZUP-35-prd 2>/dev/null || echo "Failed to get commits from origin/ZUP-35-prd"

echo ""
echo "=== Testing git log command that chr uses ==="
echo "Command: git log ^ZUP-35-hml ZUP-35-prd --format=%h|%an|%s|%ad --date=short -20"
git log ^ZUP-35-hml ZUP-35-prd --format=%h\|%an\|%s\|%ad --date=short -20 2>/dev/null || echo "FAILED: Local branch command failed"

echo ""
echo "Command: git log ^origin/ZUP-35-hml origin/ZUP-35-prd --format=%h|%an|%s|%ad --date=short -20"
git log ^origin/ZUP-35-hml origin/ZUP-35-prd --format=%h\|%an\|%s\|%ad --date=short -20 2>/dev/null || echo "FAILED: Remote branch command failed"

echo ""
echo "=== Running chr with debug output ==="
/home/carraes/projs/chr/dist/chr pick --latest --show
