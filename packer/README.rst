====================
Empire Packer Images
====================

In order to have images ready to run empire, we use packer_. This allows us
to build images in both AWS & for vagrant_/Virtualbox, in parallel, with a
single config.

-------------
Getting Setup
-------------

The entire system currently only works on MacOS hosts. This is mostly out of
laziness, but it is what it is for now.

A setup script has been provided to get you started. To use it, execute
**empire/packer/scripts/setup.sh**.

**Note**: Be sure to pay attention to the instructions that are output from
the script as there are a few manual steps that are required. If you have any
questions, ping mike on Slack.

This will install all of the software necessary to run the empire cluster in
vagrant_, including:

- packer_ (in $HOME/bin)
- awscli_ (and pip_)
- vagrant_ (plus the vagrant-vbguest plugin)
- btsync_ (we are not 100% sure we'll keep this - for now we're using it to
           provide an easy way to share images locally, so that we do not have
           to go to S3 everytime a new image is created)

-------------------
Running the Cluster
-------------------

**Note:** This will only work if you've run *setup.sh* from above, or have
installed all the necessary software manually. I suggest *setup.sh*.

In order to run the cluster, enter the **empire** repo directory and execute::

    vagrant up

If this is your first time running the command, it will take a little while
to finish setting up.  **Note:** This goes a lot faster if btsync_ was
installed via setup.sh properly, and it has finished sync'ing the appropriate
images.  If not, it will have to download the images from S3 which can take
some time.

When *vagrant up* is finished running, you should have 4 virtual machines
launched: controller and minion[1-3].

To get the status of the machines, use the following command::

    vagrant status

To connect to any of the machines, use the **vagrant ssh** command in the
**empire** repo (where the Vagrantfile is located), for example::
    
    # To connect to the controller
    vagrant ssh controller
    #  To connect to minion2
    vagrant ssh minion2


---------------
Building Images
---------------

Don't, for now.  At least not until you've talked to mike. It's a little messy,
and requires various privileges. The good news is that (hopefully) we won't
have to build new images all that often.

.. _packer: http://www.packer.io/
.. _vagrant: https://www.vagrantup.com/
.. _awscli: http://aws.amazon.com/cli/
.. _pip: https://pip.pypa.io/en/latest/
.. _btsync: http://www.getsync.com/
