import sqlite3
import json
import os
import sys
from pymongo import MongoClient, UpdateOne
import pymongo

# --- Configuration ---
MONGO_URI = "mongodb://localhost:27017/" 
MONGO_DB = "dhtcrawler"
MONGO_COLLECTION = "torrents"
BATCH_SIZE = 1000

def migrate_data(db_path):
    print(f"Migrating {db_path}...")
    sqlite_conn = sqlite3.connect(db_path)
    sqlite_conn.row_factory = sqlite3.Row
    cursor = sqlite_conn.cursor()

    mongo_client = MongoClient(MONGO_URI)
    db = mongo_client[MONGO_DB]
    collection = db[MONGO_COLLECTION]

    # Indexes
    collection.create_index("infohash", unique=True)
    collection.create_index([("name", pymongo.TEXT)])

    try:
        cursor.execute("SELECT * FROM torrents")
    except sqlite3.OperationalError:
        print(f"Skipping {db_path}: No 'torrents' table found.")
        sqlite_conn.close()
        mongo_client.close()
        return
    
    operations = []
    count = 0

    while True:
        rows = cursor.fetchmany(BATCH_SIZE)
        if not rows:
            break

        for row in rows:
            infohash = row['infohash']
            if isinstance(infohash, bytes):
                infohash = infohash.hex()
                
            try:
                files = json.loads(row['files']) if row.get('files') else []
            except (json.JSONDecodeError, TypeError):
                files = []

            mongo_document = {
                "infohash": infohash,
                "name": row.get('name', 'Unknown'),
                "length": row.get('length') or row.get('size', 0),
                "files": files,
                "discovered_at": row.get('created_at'), 
            }

            operations.append(
                UpdateOne(
                    {"infohash": infohash}, 
                    {"$set": mongo_document}, 
                    upsert=True
                )
            )

        if operations:
            try:
                collection.bulk_write(operations, ordered=False)
                count += len(operations)
                print(f"Migrated {count} records from {db_path}...")
            except Exception as e:
                print(f"Bulk write warning: {e}")
            
            operations = []

    sqlite_conn.close()
    mongo_client.close()
    print(f"Migration complete for {db_path}!")

if __name__ == "__main__":
    if len(sys.argv) > 1:
        target = sys.argv[1]
        if os.path.isdir(target):
            for f in os.listdir(target):
                if f.endswith('.db'):
                    migrate_data(os.path.join(target, f))
        elif os.path.isfile(target):
            migrate_data(target)
    else:
        print("Usage: python migrate.py <path_to_db_or_directory>")
