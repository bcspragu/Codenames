# Codenames

Codenames is an AI that plays the
[Codenames](https://en.wikipedia.org/wiki/Codenames_(board_game)) board game,
implemented in Go and utilizing the [Cloud Vision
API](https://cloud.google.com/vision/) for parsing game boards from an image,
and [word2vec](https://en.wikipedia.org/wiki/Word2vec) for hints and guesses.

## Usage

The `codenames-local` binary takes three flags:

* `--model_file` (Required) - A file containing a pre-trained word2vec model. Required, or
  the AI can't make associates between words.
* `--api_key_file` (Optional) - A file containing a Google Cloud API Key that
  has access to the Cloud Vision API. If not specified, will attempt to
  authenticate with [Google Default
  Credentials](https://developers.google.com/identity/protocols/application-default-credentials).
* `--dict_file` (Optional) - A list of newline-separated words all in capital
  letters, used to sanitize Cloud Vision output. If not specified, the image
  parser may struggle to parse a valid board.

## Models

### GoogleNews

Download the GoogleNews-nectors-negative300.bin.gz listed on https://code.google.com/archive/p/word2vec/.

It's available on [Google Drive](https://drive.google.com/file/d/0B7XkCwpI5KDYNlNUTTlSS21pQmM/edit)


### Project Gutenberg

The single file with ~30k Project Gutenberg books concatenated together is available on
[Google Drive](https://drive.google.com/open?id=1XznyDoivL3kffjL-BcNLK-BSOpJQVF1c). The file
is ~5GB gzip'd and ~15GB uncompressed. It contains ~2.3 billion words in total.

There is a [pre-trained project gutenberg model](https://drive.google.com/open?id=1Dbe5pZhN7iJsNNVXnxJ6Zo8FgS-8cHAx)
trained on this dataset available as well (400MB).

If you want to get all the data yourself and train your own model, you can follow these steps:

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


On GCE, 24cpu vs 2cpu -> ~10x improvement in speed.

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
  - 51m24s to train on 24cpu
  - 300k words/thread/sec during training
  - 399M trained binary model size (35x smaller than training data)
