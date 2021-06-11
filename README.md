# Codenames

Codenames is a hodgepodge of random components assembled over the course of
several years, mostly related to playing the
[Codenames](https://en.wikipedia.org/wiki/Codenames_(board_game)) board game.
It includes scraps of code for:

- Playing a game of Codenames alone on your local computer (lame)
- A webserver and WIP frontend for playing Codenames online with friends (kinda cool)
- An AI (based on [Word2Vec](https://en.wikipedia.org/wiki/Word2vec)) that can
  play the game, either as a Spymaster or as an Operative (pretty cool).
- Parsing images of physical game boards using the [Cloud Vision
  API](https://cloud.google.com/vision/) (cool, but why?)

Pretty much everything is written in Go.

## What's in this repo?

As noted above, this repo contains a few distinct components:

* `boardgen` - A package for generating realistic Codenames boards.
* `client` - An HTTP + WebSocket based client for the Codenames web server.
* `cmd` - Contains the entrypoints for all binaries.
  * `ai-server` - The work-in-progress implementation of an AI server, for use
    with the web service.
  * `boardgen-cli` - A command-line tool for printing out a randomized list of
    cards, can be used as input to a local game, but otherwise not that useful
  * `codenames-client` - A command-line tool for connecting to a Codenames web
    service and playing a game.
  * `codenames-local` - A command-line tool for playing a game on the
    command-line, can use AI players
  * `codenames-server` - An HTTP-based API server that can manage games in a
    SQLite database.
  * `w2v-topn` - Not sure what this was for, probably related to testing out
    our Word2Vec models.
  * `word-server` - Same deal here, some legacy code for serving up results
    from a model. Likely does a subset of what `ai-server` should do in the
    future.
* `codenames` - The package that contains all of our domain types, and an
  interface for databases to implement, which should really live in the `web`
  package, but I wrote a lot of this before I understood how to properly
  structure these things.
* `dict` - A word-lookup implementation, not sure if its actually used for
  anything.
* `game` - An implementation of the logic for playing a game of Codenames. It
  supports two methods of running, either "plug in some interfaces for players
  and run `Play()`", or get input gradually and feed it into the `Move()`
  handler.
* `httperr` - A simple helper package that simplifies our various webservers by
  allowing handlers to return errors that contain client and logging
  information.
* `hub` - A wrapper around the `gorilla/websocket` package that handles
  WebSocket-based communications between the web server and clients, where
  clients can be web-based, CLI-based, or from the AI server.
* `io` - No idea what this is, looks like it might be used as a stdin/stdout
  implementation of the Spymaster and/or Operative interfaces.
* `memdb` - An in-memory implementation of our database interface, used
  exclusively to keep tests simple.
* `sqldb` - A SQLite-based implementation of our database interface, used for
  local testing and the actual 'production' deployment.
* `vision` - Contains the code for parsing (or at least attempting to parse) a
  Codenames board from a picture.
* `web` - Contains all the handlers and logic for the Codenames web service.
  * `frontend` - Our [Next.js](https://nextjs.org/)-based frontend.


## Word2Vec Models

All the AI stuff is based on Word2Vec, this section details where those models
came from, how to download them, and how to train them.

### GoogleNews

Download the GoogleNews-nectors-negative300.bin.gz listed on
https://code.google.com/archive/p/word2vec/.

It's available on [Google
Drive](https://drive.google.com/file/d/0B7XkCwpI5KDYNlNUTTlSS21pQmM/edit)

### Project Gutenberg

The single file with ~30k Project Gutenberg books concatenated together is
available on [Google
Drive](https://drive.google.com/open?id=1XznyDoivL3kffjL-BcNLK-BSOpJQVF1c). The
file is ~5GB gzip'd and ~15GB uncompressed. It contains ~2.3 billion words in
total.

There is a [pre-trained project gutenberg
model](https://drive.google.com/open?id=1Dbe5pZhN7iJsNNVXnxJ6Zo8FgS-8cHAx)
trained on this dataset available as well (400MB).

If you want to get all the data yourself and train your own model, you can
follow these steps:

```
mkdir ~/word2vec_models

# Get the gutenberg txt data
cd ~/word2vec_models
curl http://gutenberg-tar.com/gutenberg_txt.7z
apt-get install p7zip-full
7z x gutenberg_txt.7z
tar -xf gutenberg_txt.tar

# Download/Build the word2vec project
cd ~/word2vec_models
git clone https://github.com/dav/word2vec
cd word2vec/src
make

# Make a simple model from a single book
# Note: must use -binary 1 to work with the go library
cd ~/word2vec_models/word2vec/bin
cd ./word2vec -train ~/word2vec_models/gutenberg/1/2/3/4/12345/12345.txt -output ~/word2vec_models/12345.bin -binary 1

# Clean a single file to remove extra punctuation and make everything lowercase
cat $file | tr --complement "A-Za-z'_ \n" " " | tr A-Z a-z > $file.normalized

# If you want to use word2phrase, do it before the lower case step.
# word2phrase combines words in the original text with underscores to create
# "phrases" (e.g. "We love going to New York" -> "We love going to New_York")
cd ~/word2vec_models/word2vec/bin
cat $file | tr --complement "A-Za-z'_ \n" " " > $file.phase1
./word2phrase -train $file.phase1 -output $file.phase2
cat $file.phase2 | tr A-Z a-z > $file.normalized

# Normalize and concatenate a bunch of files into a single file
# Note: the regex here only looks under the 1/1/.* directory for files; this should
# take <5mins and produce a 3.4GB text file.
# Use a more general regex (e.g. ".*/[0-9]+\.txt") to do more data.
cd ~/word2vec_models/gutenberg
time find . -regex "./1/1/.*/[0-9]+\.txt" -print0 | xargs -0 -I {} sh -c "cat {}
 | tr -c \"A-Za-z_' \n\" \" \" | tr A-Z a-z >> ~/everything.txt"
```


On GCE, 24 CPUs vs 2 CPUs -> ~10x improvement in speed.

- Training on a partial set of Project Gutenberg books
  - 3.4GB text file
  - 184K "vocab" words
  - 130M individual words
  - 3m47s to train on 24cpu
  - 265k words/thread/sec during training 
  - 74MB trained binary model size (45x smaller than training data)

- Training on a full set of Project Gutenberg books
  - 14.1GB text file (https://drive.google.com/open?id=1XznyDoivL3kffjL-BcNLK-BSOpJQVF1c)
  - 1M "vocab" words
  - 2.3B individual words
  - 51m24s to train on 24 CPUs
  - 300k words/thread/sec during training
  - 399M trained binary model size (35x smaller than training data)

### Wikipedia

1. Get an XML dump from one of the [mirror
   sites](https://dumps.wikimedia.org/mirrors.html).  [This
   one](https://wikimedia.bytemark.co.uk/) worked well for me. The XML dump
   file should be named something like "enwiki-20180201-pages-articles.xml.bz2"
   and be ~14GB (as of 2017). This is ~5M articles.

2. Now we need to convert the XML dump into a more usable format. Fortunately,
   [gensim](https://radimrehurek.com/gensim/scripts/segment_wiki.html) provides
   a great tool for this as of v3.3.0:

    ```
    pip install gensim==3.3.0
    ```

3. And now we run the tool over the bzip'd input and produce a gzip'd output
   file:

    ```
    python -m gensim.scripts.segment_wiki -f enwiki-20180201-pages-articles.xml.bz2 -o enwiki.json.gz
    ```

    This resulted in a ~6GB gzip'd file (from an original 14GB bz2'd xml file)
    and took ~4 hours to run with --workes=3 (~7K articles/minute/worker)

The output file consists of one article per line, where each line is a json
object; each object contains (among other fields):

    - title: string
    - section_title: list of strings
    - section_text: list of strings

4. Since we only care about the section_text  we can use the `smart_open`
   package to read in the gzip'd file and save only the parts we care about.

  ```
  import codecs
  import json
  import smart_open

  with codes.open('enwiki.txt', 'a+', encoding='utf_8') as output:
    for line in smart_open.smart_open('enwiki.json.gz'):
      article = json.loads(line)

      for section_title, section_text in zip(article['section_titles'], article['section_texts']):
        if section_title in ['See also', 'References', 'Further reading', 'External links', 'Footnotes', 'Bibliography', 'Notes']:
          continue
        output.write(section_text)
  ```

  This took ~15 mins and produces a ~16GB uncompressed text file with ~2.6B words

## Original Design

Here's the original design we hacked together an eternity ago.

![Picture of Whiteboard Design](whiteboard_design.jpg)

Some pieces of this are still around, some have gone the way of the dodo. In
particular, the following things exist:

- **A web app** - Should be usable on both web and mobile.
- **WebSockets** - Used for sending real-time updates from the web server to
  clients.
- **SQLite DB** - Used for persistence for the web service.
- **Docker** - The web server, Next.js frontend, and (eventually) AI server are
  packaged as Docker containers for deployment.
- **NGINX** - NGINX is no longer used to serve static assets, but is used as a
  reverse proxy to both the Next.js frontend container and the web service
  backend container.
- **Cookies** - Used for authentication, are returned as part of creating a
  user (which just requires a username).

### Basic ideas for UI Screens

This section contains some ideas for UI flows for the web client, which might
bear a passing resemblance to what's implemented in the Next.js UI.

1. Username

A user goes to https://codenames.ai/ for the first time. They enter a username.
It can have all sorts of cool emojis in it probably. We generate a cookie for
the user and persist them to our DB.

2. New Game / Join Game

A button allows the user to create a new game.

There will also be a list of names of existing games that the player can either
join (if it hasn't started yet) or spectate. Game "names" are just the IDs,
which are formed by taking three random words from the possible set of
Codenames word cards.

3. Start Game

When a new game is created, it's in a pending/lobby state. Players (including
the game's creator) can join the game at this point. The creator can then start
assigning people to roles.

Only the person who created the game can start the game.

The game won't start until all roles are filled. In the future, we'll hopefully
have the option to automatically fill any empty roles with AIs.

A spymaster can be only a single person or AI. An operative can be zero or more
humans and zero or one AIs. If there are multiple human operatives, a guess is
chosen once the majority of operatives have selected the same card. In the
future, we'd like to allow human spymasters and operatives to be able to get
"Hints" from an AI.

4. Active Game

Spymasters will have a view showing them the board with all the cards
highlighted in the right color, and indication of which words have already been
guessed, an input for their next clue, and (again, in the future) a way to get
a hint/suggestion from the AI.

Operatives will have a view showing them the board with the cards, some indication
of which words have been guessed, and the current clue. The cards will be touchable.
When a user touches a card, everyone will be able to see who touched which card.

Spectators will have a very similar view to operatives, but it will be read-only.

All the views should probably also clearly indicate who's turn it is, how many cards
each team has left. Maybe some sort of history of the game.

### Database

A SQLite database should be hilariously sufficient for our needs, and it keeps
it simple.

- User Table
    - UserId string  // related to the cookie
    - DisplayName string

- Game Table
    - GameId string  // pronounce-able
    - Status string  // enum: Pending, PLaying, PFinished
    - State blob

- GamePlayers Table
    - GameId string
    - UserId string
    - Role string  // Spymaster, Operative
    - Team string  // Red, Blue

- GameHistory Table
    - GameId string
    - EventTimestamp timestamp
    - Event blob


### Web Service API

The Web Service API is a RESTful-ish HTTP/JSON interface, with some WebSockets
sprinkled in for real-time shenanigans. More details about the API can be found
in [the web/ README](/web/README.md).

### Assorted Features and Nice-to-Haves

This is kind of a TODO section.

- Fuzzing
- AI hints for humans
- AI trash talking when humans give clues & guesses.
- A way for users to file feedback
- Spectators are called "Taters" because that's funny
- Supporting >1 human operative on a team (first come first serve for making a guess.)
- AI can play has any combination of spymasters and operatives. It could be all AIs or no AIs.
- Should like reasonable on mobile and on desktop. Would be cool to have a Cast App that shows
  a spectator screen.
