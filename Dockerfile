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
ARG SSH_PRIVATE_KEY
RUN echo "${SSH_PRIVATE_KEY}" > ~/.ssh/id_rsa && \
    chmod 600 ~/.ssh/id_rsa

# Add hosts keys to get deps
RUN ssh-keyscan github.com >> ~/.ssh/known_hosts
RUN git config --global --add url."git@github.com:".insteadOf "https://github.com/"

# get deps for both components
RUN go get github.com/teamdigitale/anpr-dashboard-server/server
RUN go get github.com/teamdigitale/anpr-dashboard-server/converter

# Build the executables.
RUN go build \
    -o server/dashboard-server ./server

RUN go build \
    -o converter/dashboard-converter ./converter

# Final stage: the running container.
FROM alpine AS final

# install certificates
RUN apk add --no-cache ca-certificates

# create destination folders
RUN mkdir -p /opt/dashboard
RUN mkdir -p /srv/anpr-dashboard

# Import the compiled executables from the second stage.
COPY --from=builder /src/server/ /srv/anpr-dashboard/server/
COPY --from=builder /src/converter/ /srv/anpr-dashboard/converter/
COPY --from=builder /src/site/ /srv/anpr-dashboard/site/

#Fix ownership to run the binary as non root
RUN chown -R nobody:nobody /srv/anpr-dashboard

#Change the work directory where server build is
WORKDIR /srv/anpr-dashboard/server/

# Declare the port on which the webserver will be exposed.
# As we're going to run the executable as an unprivileged user, we can't bind
# to ports below 1024.
EXPOSE 8443

# Perform any further action as an unprivileged user.
# USER nobody:nobody

# Run the compiled binary.
ENTRYPOINT ["./dashboard-server", "--https-listen-on=[::]:8443",\
		   "--config-file=/opt/dashboard/config.yaml",\
		   "--cookie-creds=/opt/dashboard/creds/cookie-creds.json",\
		   "--email-creds=/opt/dashboard/creds/email-creds.yaml",\
		   "--oauth-creds=/opt/dashboard/creds/oauth-creds.yaml",\
		   "--web-templates=/srv/anpr-dashboard/server/templates/",\
		   "--email-templates=/srv/anpr-dashboard/server/emails/",\
		   "--static-content=/srv/anpr-dashboard/server/static/"]

