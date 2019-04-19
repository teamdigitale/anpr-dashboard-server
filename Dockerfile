# Accept the Go version for the image to be set as a build argument.
# Default to Go 1.11
ARG GO_VERSION=1.11

# First stage: build the executable.
FROM golang:${GO_VERSION}-alpine AS builder

# Install the Certificate-Authority certificates for the app to be able to make
# calls to HTTPS endpoints.
RUN apk add --no-cache ca-certificates git openssh gcc libc-dev 

# Set the environment variables for the go command:
# * CGO_ENABLED=0 to build a statically-linked executable
# * GOFLAGS=-mod=vendor to force `go build` to look into the `/vendor` folder.
# ENV CGO_ENABLED=0 GOFLAGS=-mod=vendor

# Set the working directory outside $GOPATH to enable the support for modules.
WORKDIR /src

# Import the code from the context.
COPY ./ ./

# Add deploy key to get deps
RUN mkdir -p ~/.ssh
#ARG SSH_PRIVATE_KEY
#RUN echo "${SSH_PRIVATE_KEY}" > ~/.ssh/id_rsa && \
#    chmod 600 ~/.ssh/id_rsa

# Add hosts keys to get deps
RUN ssh-keyscan github.com >> ~/.ssh/known_hosts
#RUN git config --global --add url."git@github.com:".insteadOf "https://github.com/"

# get deps for both components
RUN go get github.com/teamdigitale/anpr-dashboard-server/server
RUN go get github.com/teamdigitale/anpr-dashboard-server/converter

# Build the executables.
RUN go build \
    -o server/dashboard ./server

RUN go build \
    -o converter/converter ./converter

# Final stage: the running container.
FROM alpine AS final

# install certificates
RUN apk add --no-cache ca-certificates sqlite

# create destination folders and external mount points
RUN mkdir -p /srv/anpr/db
RUN mkdir -p /srv/anpr/vault
RUN mkdir -p /srv/anpr/cache

# Import the compiled executables from the second stage.
COPY --from=builder /src/site/ /srv/anpr/site/
COPY --from=builder /src/server/ /srv/anpr/server/
COPY --from=builder /src/converter/ /srv/anpr/converter/

#Fix ownership to run the binary as non root
RUN chown -R nobody:nobody /srv/anpr

#Change the work directory where server build is
WORKDIR /srv/anpr/db/

# Declare the port on which the webserver will be exposed.
# As we're going to run the executable as an unprivileged user, we can't bind
# to ports below 1024.
EXPOSE 8080

# Perform any further action as an unprivileged user.
USER nobody:nobody

# Run the compiled binary.
ENTRYPOINT ["/srv/anpr/server/dashboard", "--http-listen-on=[::]:8080",\
		   "--config-file=/srv/anpr/vault/config",\
		   "--cookie-creds=/srv/anpr/vault/cookie-creds",\
		   "--email-creds=/srv/anpr/vault/email-creds",\
		   "--oauth-creds=/srv/anpr/vault/oauth-creds",\
		   "--web-templates=/srv/anpr/server/templates/",\
		   "--email-templates=/srv/anpr/server/emails/",\
		   "--static-content=/srv/anpr/server/static/"]
