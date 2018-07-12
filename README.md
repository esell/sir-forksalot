# Sir Forksalot

A great example of taking a 20 line shell script and making it into a 200 line Go program that
takes way too much work to build.

Sir Forksalot will get a list of github repos for your Github Org that are forks. From there it will iterate through them and update them with their upstream repos.


# Pre-reqs

Since I refused to "shell out" to use the actual git program, I decided to use [git2go](https://github.com/libgit2/git2go).
Also, in an attempt to make this thing somewhat portable, I did not lock in a specific [libgit2](https://github.com/libgit2/libgit2) version which means
you get to build it from source to get the latest and greatest!

On Ubuntu you can do something like:

`apt-get install libgit2-dev cmake libssl-dev libssh2-1-dev`

to get the needed tools to build libgit2. Once you have that you need to clone the libgit2 repo and follow their
build instructions to get it all built and installed.

Or you could just run the docker image and forget about all of that ;)


# Install

The usual: `go get github.com/esell/sir-forksalot`


# Usage

So now you have built way too many things that you'll never use let's run this thing!

First, set a few environment variables:


`export GITHUB_ORG=YOUR_ORG`

`export GITHUB_USERNAME=YOUR_USERNAME`

`export GITHUB_TOKEN=YOUR_PERSONAL_GITHUB_TOKEN`


Then just run it: `./sir-forksalot`
