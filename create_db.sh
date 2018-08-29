#!/bin/bash
rm -f codenames.db
sqlite3 codenames.db < sqldb/schema.sql
