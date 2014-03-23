#Getting started

**Note** -- the oti commands in this guide cannot be copy-pasted. That might
cost people money inadvertantly.  To play along get yourself the id of any AMI
in **us-east-1** (have you looked at
[packer](http://www.packer.io/intro/getting-started/build-image.html)?) and
just replace any `ami-*` values in examples below with your own image ids.  If
you complete the guide within an hour (which is easy) the cost charges from AWS
should be under $0.10 (USD).

##Essentials

The first thing to do is setup a simple configuration. Read the oticonfig
[Essentials](http://godoc.org/github.com/bmatsuo/oti/oticonfig#hdr-Essentials)
to get started with that.

##Launch instances

Once aws credentials are set up you can launch instances by manually specifying
an amazon instance id.

    $ oti launch myservice ami=ami-1a2b3d4e
    myservice:021a3dbe-2d95-4b04-bdb7-a6001cb93354
    myservice i-2b3c4d5e pending

The output line is a _session id_ that identifies all resources allocated by
`oti launch`.  The session id is used to control the lifetime of instance
groups created by `oti launch`.

In order to ssh into created instances, give oti the name of a key pair you
created previously in
[EC2](http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ec2-key-pairs.html).

    $ oti launch -keyname mysshkeypair myservice ami=ami-1a2b3d4e
    myservice:059c1003-39b8-45f4-9799-8c2be9f700e1
    myservice i-3c4d5e6f pending

Before you can ssh into the instance it has to leave the 'pending' state and initialize.

##Checking instance states

It is easy to get a high level overview of all oti sessions.

    $ oti sessions
    us-east-1	myservice:021a3dbe-2d95-4b04-bdb7-a6001cb93354	0/1/0/0/0
    us-east-1	myservice:059c1003-39b8-45f4-9799-8c2be9f700e1	1/0/0/0/0

This only shows sessions for which resources still exist in EC2.
After terminated and deleted instances are (eventually) purged oti will no longer see them.

The `oti sessions` output shows that the session we provided a `-keyname` for
is still 'pending'.  Run the command again but provide just the desired session
to filter the results

    $ oti sessions myservice:059c1003-39b8-45f4-9799-8c2be9f700e1
    us-east-1	myservice:059c1003-39b8-45f4-9799-8c2be9f700e1	0/1/0/0/0

Now it's running and you should be able to ssh into your instance.

    XXX you can't get the public DNS record for the running instances w/o using the dashboard

##Terminating instances

Now you have two "myservice" sessions running. You can terminate one of them

    $ oti terminate myservice:021a3dbe-2d95-4b04-bdb7-a6001cb93354
    i-2b3c4d5e shutting-down (was running)

Or you can terminate all "myservice" sessions.

    $ oti terminate -s myservice
    i-3c4d5e6f shutting-down (was running)

