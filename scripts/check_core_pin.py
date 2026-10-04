"""Check the reference deployment's independent headless core receipt and pin."""
import json
from pathlib import Path

from core_release import validate_core


def main():
    root = Path(__file__).resolve().parents[1]
    receipt = json.loads((root / 'deploy/core-release.json').read_text())
    consumer = {'mode': 'api', 'adoption': 'core', 'rollout': 'reviewed-pr',
                'media': ['print', 'audio', 'video'], 'checks': [['make', 'verify-core']]}
    validate_core(receipt, consumer, receipt)
    pin = (root / 'deploy/archive-core.env').read_text()
    if pin != 'ARCHIVE_ENGINE_IMAGE=' + receipt['engine_digest'] + '\n':
        raise SystemExit('Core pin differs from the reviewed independent release')
    print('Independent headless core pin verified:', receipt['version'], receipt['source_revision'])


if __name__ == '__main__':
    main()
