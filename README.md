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
