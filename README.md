# DDNS Updater
This is a needlessly advanced program designed to be used to update Namecheap
DDNS entries. Feel free to throw it on your local server or NAS to keep your
DNS entries pointing at your home.

By default, the program will use a small cache file to store the current public
IP address of the machine. This is to reduce the number of requests made to the
Namecheap servers. It will also use an internal list of ip address resolvers
to attempt to detect the public ip address of the local machine. By default it
will use `http://icanhazip.com` then `http://checkip.amazonaws.com`, however,
these resolvers can be configured through command line flags or through a
configuration file.

## Usage
### Simple
The simplest way to use this binary is by configuring a cron job (or windows
scheduled task) to run the program like below:
```
> ddns-updater -d <namecheap domain> -n <namecheap host> -t <namecheap token>
```

You can learn how to set up DDNS for a namecheap domain [here](https://www.namecheap.com/support/knowledgebase/article.aspx/595/11/how-do-i-enable-dynamic-dns-for-a-domain/).
You want to use the "Dynamic DNS Password" as the "token" for the binary.

Example (this will update `home.example.com` to the public ip address of the 
machine running the program)
```
> ddns-updater -d example.com -n home -t 1234abcd5789efgh
```

### Configuration file
If you wish to use a configuration file place a file named `.ddns-updater.yaml`
in either the directory of the binary or in your systems `$HOME` directory. The
file can configure the following values:

```yaml
namecheap:
  host: home
  domain: example.com
  token: 123abc
ipResolvers:
  - "http://otheripresolver.com"
  - "http://otherotheripresolver.com"
cacheFile: "~/.configFiles/cache-file"
```

## FAQ
    What if I don't want to cache my IP? I am paying to use Namecheap's service so
    I am going to use the whole service.

Thats ok, I am not here to police your API usage. Just set the `cacheFile` to `""`
in the config file or through the command line flag.


    I want to use my own IP resolver. How do I need to set it up for it to work with
    this program?

The program expects the IP resolvers to simply return the IP address in
plaintext when they are hit with a HTTP GET request. Don't worry about trailing
whitespace, the program will strip out everything around the IP address.

