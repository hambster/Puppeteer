Puppeteer
=========

URL screenshot server powered by PhantomJS and Go.

## Requirements

* [PhantomJS](http://www.phantomjs.org/)  
* Linux x64  
* Optional: to build Puppeteer, you need [Go](http://golang.org)  

## Installation

For installation, just copy puppeteer binary from **release** directory.

## Usage

Currently, Puppeteer provides 2 executables:

* **puppeteer**: daemon process to take screenshot.  
* **puppeteer-web**: http server to accept http request, and save request as file for puppeteer.

both puppeteer and puppeteer-web take **puppeteer.conf** as command argument.
You can run puppeteer and puppeteer-web like:  

_puppeteer puppeteer.conf&_
_puppeteer-web puppeteer.conf&_

## Project Status

Puppeteer is feature complete currently.  
You can use it to take screenshot now.  
Advanced features will be added in the future.  

### The Web API Protocols

* GET /info/{key}  
  To get information about specific screenshot key.  
  The respnonse will be JSON format. The detail of  
  the JSON format are as follows:  

        {
            "RetCode": $retCode,          //int, return code
            "RetMsg": "$retMsg",          //string, message about return code
            "Data":{
                "Key": "$key",            //string, request key associate with screenshot.  
                "Status": $status,        //int, 1 for ready,
                                          //     2 for running,
                                          //     3 for not exists
                "LastUpdate": $timestamp  //int, timestamp of screenshot last update time.
            }
        }

* POST /info/  
  To generate screenshot with given POST parameters:  

      - url: to url to take screenshot.  
      - userAgent: to user agent to include in request header.    

    The response will be JSON format. The detail of  
    the JSON format are as follows:

        {
            "RetCode": $retCode,          //int, return code. 0 for success.
            "RetMsg": "$retMsg",          //string, message about return code
            "Data":{
                "Key": "$key",            //string, request key associate with given url.
                                          //        used for subsequent /info/ and /pic/ API request.
                "Status": $status,        //int, 1 for ready,
                                          //     2 for running,
                                          //     3 for not exists
                "LastUpdate": $timestamp  //int, timestamp of screenshot last update time.
            }
        }

* GET /pic/{key}  
  To download screenshot as inline images (i.e., you can use  
  this url in html &lt;img&gt; directly.) Please check HTTP response code.
  For valid screenshot, you will get:
  
  * **Status 200** with **Content-Disposition: inline; filename=screenshot.png**.  

    For invalid screenshot, you will get **Status 404** or other HTTP response code.

## History

* v0.5: Initial feature complete version.
