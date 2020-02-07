# Get the latest alpine with curl installed
FROM appropriate/curl:3.1

# Import the script from the context.
COPY fetch-data.sh .

#Fix ownership to run the script as non root
RUN chown -R nobody:nobody fetch-data.sh

# Perform any further action as an unprivileged user.
USER nobody:nobody

ENTRYPOINT ["bin/sh"]

CMD ["-c", "source fetch-data.sh"]
