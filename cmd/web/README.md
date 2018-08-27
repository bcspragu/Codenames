# Codenames Web

This directory holds the Codenames web server and frontend. Currently, it's
just the frontend, but that will probably change at some point, we'll see.

## Running the frontend

To avoid installing JavaScript garbage on my beautiful computer, I run the
frontend in Docker. To run the frontend in development mode with live reloading
and linting and stuff, first build the Docker container, which is just a simple
Alpine image containing `npm` and `yarn`, then run the container.

```bash
# Builds the container.
$ ./build.sh
$ cd frontend
# You only need to run install.sh once.
$ ./install.sh
$ ./serve.sh
```

The server should be running at `localhost:8080`
