from pathlib import Path
import secrets
p = Path('.env')
if p.exists():
    print('.env already exists; preserving your configuration.')
else:
    text = Path('.env.example').read_text()
    text = text.replace('replace-with-at-least-32-random-bytes-before-running', secrets.token_urlsafe(48))
    p.write_text(text)
    p.chmod(0o600)
    print('Created .env with a random local JWT secret.')
