#!/usr/bin/env python3
"""Derive a collection-only deployment dump from a retained recovery checkpoint."""
import argparse
import json
from pathlib import Path
import re
import subprocess
import time
from recovery import run, capture, checksum

def main():
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--snapshot',required=True)
    parser.add_argument('--collection',required=True)
    parser.add_argument('--output',required=True)
    args=parser.parse_args()
    if not re.fullmatch(r'[A-Za-z0-9][A-Za-z0-9._-]{0,199}',args.collection): raise RuntimeError('Invalid collection identity')
    snapshot=Path(args.snapshot).resolve(); output=Path(args.output).resolve()
    if output.exists(): raise RuntimeError('Preserve an existing export; choose a new output path')
    output.parent.mkdir(parents=True,exist_ok=True,mode=0o700)
    name='vhness-export-'+args.collection.lower().replace('_','-')+'-'+str(time.time_ns())
    run(['docker','run','-d','--name',name,'--network','none','--memory','768m','--cpus','1','-e','POSTGRES_HOST_AUTH_METHOD=trust','postgres:17'],stdout=subprocess.DEVNULL)
    try:
        for _ in range(60):
            ready=subprocess.run(['docker','exec',name,'pg_isready','-h','127.0.0.1','-U','postgres'],stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
            if ready.returncode==0:break
            time.sleep(.5)
        else:raise RuntimeError('Owned export database did not become ready')
        with snapshot.open('rb') as stream:run(['docker','exec','-i',name,'pg_restore','-U','postgres','-d','postgres','--exit-on-error','--no-owner','--no-acl'],stdin=stream)
        collection=args.collection
        # All removal is confined to this disposable derived copy. The original
        # dump, active databases, source metadata, and media are never altered.
        query=f"""BEGIN;
CREATE TEMP TABLE keep_items AS SELECT id AS item FROM episodes WHERE collection_id='{collection}' UNION SELECT item_id FROM metadata_queue WHERE collection_id='{collection}' UNION SELECT source_item FROM collections WHERE id='{collection}' AND source_item<>'';
DELETE FROM current_schedule WHERE revision IN (SELECT revision FROM schedules s WHERE EXISTS(SELECT 1 FROM jsonb_array_elements(s.payload->'programs') p WHERE NOT EXISTS(SELECT 1 FROM episodes e WHERE e.id=p->>'episode_id' AND e.collection_id='{collection}')));
DELETE FROM schedules s WHERE EXISTS(SELECT 1 FROM jsonb_array_elements(s.payload->'programs') p WHERE NOT EXISTS(SELECT 1 FROM episodes e WHERE e.id=p->>'episode_id' AND e.collection_id='{collection}'));
DELETE FROM decision_history WHERE episode_id IN (SELECT id FROM episodes WHERE collection_id<>'{collection}');
DELETE FROM preservation_history WHERE episode_id IN (SELECT id FROM episodes WHERE collection_id<>'{collection}');
DELETE FROM episodes WHERE collection_id<>'{collection}';
DELETE FROM metadata_queue WHERE collection_id<>'{collection}';
DELETE FROM worker_state WHERE collection_id<>'{collection}';
DELETE FROM collection_budget_reviews WHERE collection_id<>'{collection}';
DELETE FROM collections WHERE id<>'{collection}';
DELETE FROM observations o WHERE NOT EXISTS(SELECT 1 FROM keep_items k WHERE k.item=o.item_id);
DELETE FROM snapshots s WHERE NOT EXISTS(SELECT 1 FROM keep_items k WHERE k.item=s.item_id) AND NOT EXISTS(SELECT 1 FROM episodes e WHERE e.source_hash=s.sha256);
COMMIT;"""
        run(['docker','exec',name,'psql','-U','postgres','-d','postgres','-v','ON_ERROR_STOP=1','-c',query],stdout=subprocess.DEVNULL)
        stats=capture(['docker','exec',name,'psql','-U','postgres','-d','postgres','-At','-c',"SELECT (SELECT count(*) FROM collections),(SELECT count(*) FROM episodes),(SELECT count(*) FROM snapshots),(SELECT count(*) FROM snapshots WHERE encode(sha256(raw),'hex')<>sha256)"],text=True).strip().split('|')
        if stats[0]!='1' or stats[3]!='0':raise RuntimeError('Export scope or source-byte verification failed')
        with output.open('wb') as stream:run(['docker','exec',name,'pg_dump','-U','postgres','-d','postgres','-Fc'],stdout=stream)
        output.chmod(0o600)
        receipt={'collection':collection,'source_dump_sha256':checksum(snapshot),'export_sha256':checksum(output),'bytes':output.stat().st_size,'episodes':int(stats[1]),'snapshots':int(stats[2]),'invalid_raw_snapshots':int(stats[3]),'boundary':'Collection-only derived recovery copy; no original metadata or media removed'}
        output.with_suffix('.receipt.json').write_text(json.dumps(receipt,indent=2)+'\n');output.with_suffix('.receipt.json').chmod(0o600)
        print(json.dumps(receipt))
    finally:run(['docker','rm','-f','-v',name],stdout=subprocess.DEVNULL)

if __name__=='__main__':main()
