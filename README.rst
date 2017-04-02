STS
===

| STS is a command-line application to obtain temporary credentials via the AWS (Amazon Web Services) **S**\ ecurity **T**\ oken **S**\ ervice.  
| It's intended to be a drop-in replacement for the functionality provided by the official awscli application: https://aws.amazon.com/cli/ .  

Benefits
--------
- Option to drop into a shell with the temporary credentials already set
- Automatically acquire MFA device serial (via IAM request and checking environment variables)
- Option to unset AWS environment variables first, so that subsequent calls for get-session-token etc. don't attempt to use existing temporary credentials
- Native binary available, so simpler installation with no dependencies

Known issues
------------
- STS currently assumes you are using an MFA device, so always requests a token code. If MFA isn't activated, then hit <enter> to bypass 'Enter token value:' prompt.
- assume-role-with-saml not yet implemented
- assume-role-with-web-identity not yet implement
- minimal testing on Windows

Roadmap
-------
- proper documentation
- bash/zsh completion

Installation
------------

Download the latest release for your OS and architecture from: https://github.com/jonhadfield/sts/releases.

For example, on 64-bit Linux:
::

    curl -L "https://github.com/jonhadfield/sts/releases/download/1.1.0/sts_linux_amd64" -o /usr/local/bin/sts ; chmod +x /usr/local/bin/sts

on MacOS:
::
    curl -L "https://github.com/jonhadfield/sts/releases/download/1.1.0/sts_darwin_amd64" -o /usr/local/bin/sts ; chmod +x /usr/local/bin/sts


Example Usage
-------------

Note: In order to get temporary credentials, you must first provide your permanent credentials as detailed `here
<http://docs.aws.amazon.com/sdk-for-java/v1/developer-guide/credentials.html>`_.

Obtain temporary credentials for a user:
::

    sts get-session-token --serial-number arn:aws:iam::123456789012:mfa/user --token-code 123456

Obtain temporary credentials for an assumed role:
::

    sts assume-role --role-session-name myrolesession --role-arn arn:aws:iam::123456789012:role/myrole
