var page = require('webpage').create();
var system = require('system');
var url = system.args[1];
var output = system.args[2];

page.open(url, function() {
    page.render(output);
    page.close();
    phantom.exit();
});

