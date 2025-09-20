<p align="center">
    <img style="width:8em;" src="./assets/logo.png" alt="jim">
</p>

# brodo

`br`anch t`odo` is a way to find out what needs to be done in your branch before is ready to be merged

usage:

```bash
cd /path/to/your/project
git checkout your-branch
brodo
brodo --pattern '// TODO:\ '
brodo --pattern '// TODO:\ ' <branch> # add a branch if no PR exists
```

## Install

```bash
go install github.com/just-hms/brodo@latest
```
