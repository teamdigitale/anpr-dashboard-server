# What is this?

This directory contains the web pages for the ANPR dashboard.

The git repository should contain all necessary files without need for you
to do anything. It's just a plain old static site.
The instructions below are for developers only.


# Developers' instructions

## Setup the machine to run the site

It's just static html pages right now, just configure your favourite
web server as you wish.

## Setup your machine to change the web site

1) Install npm. On a Debian derived system:

    apt-get install npm
    npm install -g npm

2) Install bower-installer:

    npm install bower-installer

## Update / fetch the libraries used by this web site

3) Finally fetch all the dependencies:

    bower-installer

## Generating (or updating) the data needed

Just go in ../converter, and follow the instructions there.

This directory uses symlinks into ../converter, so everything
should just work out of the box.

## API Keys

The google maps APIs require a key to run. The key is currently
restricted to only work from the IP of our web server.

Unfortunately, the key is public - it MUST be in the HTML page.

To test, generate your own key, and replace it in the file:
https://developers.google.com/maps/documentation/geocoding/intro

(I know, it is annoying - do you have a better suggestion for static pages?)

## Run the web service on your test machines 

Just use something like:

    python -m SimpleHTTPServer 

