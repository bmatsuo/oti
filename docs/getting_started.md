#Quick Start

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

Once aws credentials are set up, `oti launch` can launch instances by
specifying an amazon instance id.

    $ oti launch myservice ami=ami-1a2b3d4e
    myservice:021a3dbe-2d95-4b04-bdb7-a6001cb93354
    myservice i-2b3c4d5e pending

The output line is a _session id_ that identifies all resources allocated by
`oti launch`.  The session id is used to control the lifetime of instance
groups created by `oti launch`.

##Checking instance states

It is easy to get a high level overview of all oti sessions.

    $ oti sessions
    us-east-1	myservice:021a3dbe-2d95-4b04-bdb7-a6001cb93354	1/0/0/0/0

This only shows sessions for which resources still exist in EC2.  After
terminated and deleted resources are (eventually) purged oti will no longer see
them.

This shows that the "myservice" session just started has 1 instance in the
"us-east-1" region in the "pending" state.

##Terminating instances

With a "myservice" session running, `oti terminate` can terminate it.

    $ oti terminate -s myservice
    i-3c4d5e6f shutting-down (was running)

This command looks for all sessions with the "myservice" type and terminates
their instances.  There was only one "myservice" session running so it doesn't
matter.  But it's worth remembering when using the command.

#Connecting to your instances (using ssh)

In order to ssh into created instances you'll need to create a [key
pair](http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ec2-key-pairs.html) in
EC2 and a [security
group](http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/using-network-security.html)
that allows incoming ssh connections (port 22). Give the key pair and security
group names to oti and it will add them to your instance.

    $ oti launch -keyname=mykp -secgroup=mysg myservice ami=ami-1a2b3d4e
    myservice:059c1003-39b8-45f4-9799-8c2be9f700e1
    myservice i-3c4d5e6f pending

Before you can ssh into the instance it has to leave the 'pending' state and
initialize.

The `oti sessions` command from the [Quick Start](#Quick_Start) can tell you
when the session's instances are "running".  But you also need to the
instance's public DNS name to give the ssh command.  The `oti instances`
command gives more detailed information about the instances belonging to a
session.

    $ oti instances myservice:059c1003-39b8-45f4-9799-8c2be9f700e1
    us-east-1   i-3c4d5e6f  running ec2-54-198-39-32.compute-1.amazonaws.com

If instance is `running` you will see its public DNS name next to it's status.
You can now ssh into the instance using the private key associated with the EC2
key pair used to launch the session.

    $ ssh -i /path/to/mykp.pem ubuntu@ec2-54-198-39-32.compute-1.amazonaws.com
    ...
    ubuntu:~$

When you are done, terminate the instance.

    $ oti terminate -s myservice
    i-3c4d5e6f shutting-down (was running)

Default key pairs and security groups can be declared in the
[oticonfig](http://godoc.org/github.com/bmatsuo/oti/oticonfig#Ec2Region) file.
For details of this example, see the [example oticonfig](#example-oticonfig).

#Tagging images

Without a configuration file oti requires image ids to be explicitly given to
the lanch command.  If you have image tags configured in your
[oticonfig](http://godoc.org/github.com/bmatsuo/oti/oticonfig#Images) oti will
attempt to guess which image to launch when you don supply an explicit image id.

    $ oti launch myservice
    myservice:475c0202-0c3c-4249-81bd-aae0b783c61d
    myservice i-0f9e8d7c pending
    $ oti terminate -s myservice
    i-5fbb487e shutting-down (was pending)

With the [example oticonfig](#example-oticonfig) `oti launch` looks for images
with a "Name" tag equal to "myservice".  If multiple "myservice" images are
found then the image with the most recent "BuildTime" tag holding the most
recent timestamp is selected.

#Example oticonfig

To complete the guide, the example configuration described above is presented here.

    {
        "Ec2": {
            "Regions": [
                "RegionName": "us-east-1",
                "KeyName": "mykp",
                "SecurityGroups": [{"name":"ssh-only"}]
            ]
        },
        "Images": {
            "NameTag": "Name",
            "BuildDateTag": "BuildTime"
        }
    }
