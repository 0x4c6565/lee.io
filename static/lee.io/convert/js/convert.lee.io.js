var app = new Vue({
  el: '#app',
  data: {
    lhs: '',
    rhs: '',
    type: '',
    tools: {},
    rhsReadOnly: true,
    lhsHasError: false,
    rhsHasError: false,
    lhsFired: false,
    rhsFired: false,
    lock: false,
  },
  watch: {
    lhs: function (val) {
      if (this.rhsFired) {
        this.rhsFired = false;
        return;
      }

      this.lhsFired = true;
      this.convertCurrentTool(val, false);
    },
    rhs: function (val) {
      if (this.lhsFired) {
        this.lhsFired = false;
        return;
      }

      this.rhsFired = true;
      this.convertCurrentTool(val, true);
    },
    type: function () {
      tool = this.getCurrentTool();
      if (tool !== undefined) {
        if (tool.rhsEditable) {
          this.rhsReadOnly = false;
        } else {
          this.rhsReadOnly = true;
        }

        this.convert(tool, this.lhs, false);
      }
    }
  },
  mounted: function() {
    var t = HashParameters.get("t");
    if (t !== undefined) {
      this.type = t;
    }
  },
  methods: {
    clearError: function () {
      if (this.lhsHasError || this.rhsHasError) {
        toastr.clear();
        this.lhsHasError = false;
        this.rhsHasError = false;
      }
    },
    convert: function (tool, val, rhs) {
      try {
        if (this.lock) {
          return;
        }

        this.lock = true;

        if (rhs && !tool.rhsEditable) {
          return;
        }

        val = tool.func(val, rhs);
        this.clearError();

        this.setValue(val, rhs);
      } catch (e) {
        this.addError(e.toString(), rhs);
      } finally {
        this.lock = false;
      }
    },
    convertCurrentTool: function (val, rhs) {
      tool = this.getCurrentTool();
      if (tool !== undefined) {
        return this.convert(tool, val, rhs);
      }
    },
    toolExists: function (name) {
      return (this.getTool(name) !== undefined);
    },
    getTool: function (name) {
      if (name in this.tools) {
        return this.tools[name];
      }

      return undefined;
    },
    getCurrentTool: function () {
      return this.getTool(this.type);
    },
    addTool: function (name, displayName, func, rhsEditable = false) {
      Vue.set(this.tools, name, {
        displayName: displayName,
        func: func,
        rhsEditable: rhsEditable
      });
    },
    setValue: function (val, rhs) {
      if (val === undefined) {
        return;
      }

      if (rhs) {
        this.lhs = val;
      } else {
        this.rhs = val;
      }
    },
    addError: function (error, rhs = null) {
      toastr.clear();
      toastr.error(error, null, { timeOut: 0, extendedTimeOut: 0 });
      if (rhs === true) {
        this.rhsFired = false;
        this.rhsHasError = true;
      } else {
        this.lhsFired = false;
        this.lhsHasError = true;
      }
    }
  }
})

app.addTool("base64", "To/From Base64", function (val, rhs) {
  if (rhs) {
    return atob(val);
  } else {
    return btoa(val);
  }
}, true);

app.addTool("bytestostring", "Bytes to String", function (bytes) {
  if (isNaN(bytes)) {
    throw "Invalid number";
  }

  if (bytes == 0) {
    return '0 Bytes';
  }

  var k = 1024;
  var dm = 2;
  var sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB', 'PB', 'EB', 'ZB', 'YB'];
  var i = Math.floor(Math.log(bytes) / Math.log(k));

  if (i > (sizes.length - 1)) {
    throw "Number too large";
  }

  return parseFloat((bytes / Math.pow(k, i)).toFixed(dm)) + ' ' + sizes[i];
});

app.addTool("encodedecodeuricomponent", "Encode/Decode URI Component", function (val, rhs) {
  if (rhs) {
    return decodeURIComponent(val);
  } else {
    return encodeURIComponent(val);
  }
}, true);

app.addTool("encodedecodeuri", "Encode/Decode URI", function (val, rhs) {
  if (rhs) {
    return decodeURI(val);
  } else {
    return encodeURI(val);
  }
}, true);

app.addTool("tofromuppercase", "To/From Uppercase", function (val, rhs) {
  if (rhs) {
    return val.toLowerCase();
  } else {
    return val.toUpperCase();
  }
}, true);

app.addTool("sortalphabeticalasc", "Sort Alpabetical (Ascending)", function (val) {
  var arr = val.split("\n");
  arr.sort(function (a, b) {
    var textA = a.toUpperCase();
    var textB = b.toUpperCase();
    return (textA < textB) ? -1 : (textA > textB) ? 1 : 0;
  });

  return arr.join("\n");
});

app.addTool("sortalphabeticaldesc", "Sort Alpabetical (Descending)", function (val) {
  var arr = val.split("\n");
  arr.sort(function (a, b) {
    var textA = a.toUpperCase();
    var textB = b.toUpperCase();
    return (textA > textB) ? -1 : (textA < textB) ? 1 : 0;
  });

  return arr.join("\n");
});

app.addTool("sortnumericasc", "Sort Numeric (Ascending)", function (val) {
  var arr = val.split("\n");
  arr.sort(function (a, b) {
    return a - b;
  });

  return arr.join("\n");
});

app.addTool("sortnumericdesc", "Sort Numeric (Descending)", function (val) {
  var arr = val.split("\n");
  arr.sort(function (a, b) {
    return b - a;
  });

  return arr.join("\n");
});

app.addTool("unixtimestamp", "From/To Unix Timestamp", function (val, rhs) {
  if (rhs) {
    if (val == '') {
      val = new Date().toISOString();
    }
    var newVal = Math.round((new Date(val).getTime() / 1000));
    if (isNaN(newVal)) {
      throw "Invalid date";
    }

    return Math.round((new Date(val).getTime() / 1000));
  } else {
    if (val == '') {
      val = new Date().getTime() / 1000;
    }
    return new Date(val * 1000).toISOString();
  }
}, true);

app.addTool("sanitizemac", "Sanitize MAC Address", function (val) {
  if (val == '') {
    throw "empty mac";
  }
  val = val.replace(/-/g, '').replace(/:/g, '');

  if (val.length != 12) {
    throw "invalid mac";
  }

  return val.match(/.{2}/g).join(':');
});

app.addTool("minifyjson", "Minify JSON", function (val) {
  if (val == '') {
    return 'null';
  }

  return JSON.stringify(JSON.parse(val));
});

app.addTool("mysqlpassword", "To MySQL Password", function (val) {
  if (val == '') {
    return '';
  }

  return ("*" + CryptoJS.SHA1(CryptoJS.SHA1(val))).toUpperCase();
});

app.addTool("jsontocsharp", "JSON to C#", function (val) {
  if (val == '') {
    return '';
  }

  var converter = new json2csharp();
  return converter.renderObject(JSON.parse(val));
});

app.addTool("jsontogo", "JSON to Go", function (val) {
  if (val == '') {
    return '';
  }

  var converter = new json2go();
  return converter.renderObject(JSON.parse(val));
});

app.addTool("binarytohex", "Binary to Hex", function (val) {
  return parseInt(val, 2).toString(16);
});

app.addTool("stringtohex", "String to Hex", function (val) {
  var result = "";
  for (i = 0; i < val.length; i++) {
    result += val.charCodeAt(i).toString(16);
  }

  return result;
});

app.addTool("yamljson", "YAML <> JSON", function (val, rhs) {
  if (rhs) {
    return YAML.stringify(JSON.parse(val), 20, 2);
  } else {
    return JSON.stringify(YAML.parse(val), null, 2);
  }
}, true);

app.addTool("tabstospaces", "Tabs to Spaces", function (val) {
  return val.replace(/\t/g, "    ");
});

app.addTool("topascalcase", "To PascalCase", function (val) {
  // var result = "";
  // var nextUpper = true;
  // for (var i = 0; i < val.length; i++) {
  //   var char = val.charAt(i);
  //   var match = char.match(/([\s_-])/)
  //   if (match != null) {
  //     nextUpper = true;
  //     continue
  //   }

  //   if (nextUpper) {
  //     result.concat(char.toUpperCase());
  //     nextUpper = false;
  //   } else {
  //     result.concat(char.toLowerCase());
  //   }
  // }
  // return result;



  return `${val}`
    .replace(new RegExp(/[-_]+/, 'g'), ' ')
    .replace(new RegExp(/[^\w\s]/, 'g'), '')
    .replace(
      new RegExp(/\s+(.)(\w+)/, 'g'),
      ($1, $2, $3) => `${$2.toUpperCase() + $3.toLowerCase()}`
    )
    .replace(new RegExp(/\s/, 'g'), '')
    .replace(new RegExp(/\w/), s => s.toUpperCase());
});

app.addTool("tocamelcase", "To camelCase", function (val) {
  return val.toLowerCase()
    .replace(/['"]/g, '')
    .replace(/\W+/g, ' ')
    .replace(/_/g, ' ')
    .replace(/ (.)/g, function ($1) { return $1.toUpperCase(); })
    .replace(/ /g, '');
});

app.addTool("tosha1", "To SHA1", function (val) {
  return CryptoJS.SHA1(val).toString();
});

app.addTool("tosha256", "To SHA256", function (val) {
  return CryptoJS.SHA256(val).toString();
});

app.addTool("tosha512", "To SHA512", function (val) {
  return CryptoJS.SHA512(val).toString();
});

app.addTool("prettifyyaml", "Prettify YAML", function (val) {
  return YAML.stringify(YAML.parse(val), 100, 2);
});

app.addTool("prettifyjson", "Prettify JSON", function (val) {
  if (val == '') {
    val = null;
  }
  return JSON.stringify(JSON.parse(val), null, 2);
});

app.addTool("prettifyxml", "Prettify XML", function (val) {
  return vkbeautify.xml(val, 4);
});

app.addTool("prettifysql", "Prettify SQL", function (val) {
  return sqlFormatter.format(val, { language: "sql" });
});

app.addTool("prettifyjs", "Prettify Javascript", function (val) {
  return js_beautify(val);
});

app.addTool("prettifycss", "Prettify CSS", function (val) {
  return css_beautify(val);
});

app.addTool("fromx509", "From x509 Certificate", function (val) {
  if (val == '') {
    return '';
  }

  let certData = '';
  $.ajax({
    url: '/ssldecode',
    type: 'POST',
    dataType: 'text',
    async: false,
    data: val,
    error: function (jqXHR, textStatus) {
      throw ('Failed to decode certificate');
    },
    success: function (data) {
      certData = data;
    }
  });

  return certData;
});

app.addTool("tocontrolcharacters", "String to control characters", function (val) {
  return val.replace(/\\r\\n/, '\r\n').replace(/\\n/g, '\n').replace(/\\t/, '\t');
});

app.addTool("base64toimage", "Base64 to image", function (val) {
  var image = new Image();
  image.src = 'data:image/png;base64,iVBORw0K...';
  return image;
});
