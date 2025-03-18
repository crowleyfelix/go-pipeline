# Contributing

We use `make` to automate some useful tasks of the project. The Makefile in the root points to Makefile available in inner folders, you only need to use the prefix of the context to access tasks from the target Makefile, **when running `make` in the root folder**.

```shell
make test 

# [test] is the task available in the root Makefile.
```

```shell
make docker-up 

# [docker] is the context.
# [up] is the task available in the Makefile in the docker folder.
```

Makefile availables:

- [root](../Makefile)
- [docker](../docker/Makefile)

## Run locally

### Requirements

- [make](https://www.gnu.org/software/make/manual/make.html)
- [go1.21](https://go.dev/doc/install)

### Initialize environment

```shell
make init
```

The application and dependencies containers will start.

You just need to execute the previous command once. To start the application in the next time use the following command:

```shell
make docker-up run
```

### Execute tests

```shell
make test
```

- more commands: [Makefile](./Makefile)
- more info: <https://medium.com/@tim_raymond/fetching-private-dependencies-with-go-modules-1d65afe47c62>
