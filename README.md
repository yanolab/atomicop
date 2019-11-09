# atomicop
![cicleci](https://img.shields.io/circleci/project/github/yanolab/atomicop.svg?label=circleci&style=popout)
![license](https://img.shields.io/github/license/yanolab/atomicop.svg?style=popout)
![goversion](https://img.shields.io/badge/Go-1.12-green.svg)

atomicop supports atomic operation on various environment such as local, cloud.
The user should implement SyncRepository.
```
// SyncRepository is a repository
// This repository assure that key is already exists or not.
// If key is already exists, Store must return an error which has Duplicate() bool method.
type SyncRepository interface {
	Store(ctx context.Context, key string) error
}
```
If you use SyncMapRepository, your operation will be atomic on your process.
If you use MySQLRepository, your operation will be atomic on between using MySQL.

In addition, this library supports retryable oncer.
That means if some function is failed to execute and returns retryable error, the function can be execute again.
However you should implement retryable error correctly.
That error should have `CanRetry() bool` method.
And you should implement StateRepository in addition.
```
// StateRepository is an interface for retryable oncer
type StateRepository interface {
	// GetState gets state.
	GetState(ctx context.Context, key string) (*State, error)
	// UpdateState update state.
	UpdateState(ctx context.Context, key string, state State) error
}
```

# How to run tests
First, run docker-compose
```bash
cd docker
docker-compose up
```

Then, run go test
```bash
go test -v -race ./... -cover
```
