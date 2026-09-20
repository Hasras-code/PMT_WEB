import re
import sys
from pathlib import Path
name = sys.argv[1] if len(sys.argv) > 1 else ''
if not re.fullmatch(r'[a-z][a-z0-9_]*', name):
    sys.exit('Usage: make migration name=descriptive_name')
root = Path('migrations')
number = max(int(p.name.split('_')[0]) for p in root.glob('*.up.sql')) + 1
for direction in ('up', 'down'):
    (root / f'{number:06}_{name}.{direction}.sql').write_text('-- Write migration here.\n')
