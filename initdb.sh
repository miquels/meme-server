#! /bin/sh

DIR=/home/mikevs/data/db

rm -f $DIR/ripememes.db
sqlite3 $DIR/ripememes.db <<EOF

create table characters(
	id		INTEGER PRIMARY KEY AUTOINCREMENT,
	created		INTEGER,
	ipaddress	INTEGER,
	name		TEXT,
	rating		INTEGER,
	filename	TEXT
);

create table images(
	id		INTEGER PRIMARY KEY AUTOINCREMENT,
	created		INTEGER,
	ipaddress	TEXT,
	usercookie	TEXT,
	character_id	INTEGER,
	rating		INTEGER,
	text1		TEXT,
	text2		TEXT,
	deleted		INTEGER
);

CREATE TABLE ratedimages(
	id		INTEGER PRIMARY KEY,
	clientuuid	TEXT,
	image_id	INTEGER,
	delta		INTEGER
);
CREATE UNIQUE INDEX ratedimages_idx ON ratedimages(clientuuid, image_id);

CREATE TABLE ratedcharacters(
	id		INTEGER PRIMARY KEY,
	clientuuid	TEXT,
	character_id	INTEGER,
	delta		INTEGER
);
CREATE UNIQUE INDEX ratedcharacters_idx ON ratedcharacters(clientuuid, character_id);

EOF

chmod 666 $DIR/ripememes.db

