var logs;

function connect_log_socket(display, path) {
  logs = new WebSocket('ws://' + location.host + path);

  logs.onopen = function(e) {
    $(display).text('');
  }

  logs.onclose = function(e) {
    console.log('onclose', e);
    //setTimeout(connect_log_socket, 1000);
  }

  logs.onmessage = function(e) {
    $(display).append(e.data);
    $(display)[0].scrollTop = $(display)[0].scrollHeight;
  }
}

function table_scroll(table, height) {
  var widths = [];

  var clone = $(table).clone().appendTo('body > .container');
  var width_no_scroll = clone.width();

  clone.wrap('<div style="max-height:'+height+'; overflow-y:scroll;"></div>');
  var wrap = clone.parent();
  var width_scroll = clone.width();

  var tr = clone.find('tbody > tr')[0];

  if (tr) {
    $(tr).find('td').each(function() {
      widths.push($(this).width());
    });
  }

  // lengthen last th by the scrollbar size
  widths[widths.length-1] += (width_no_scroll - width_scroll + 2);

  wrap.remove();

  $(table).addClass('scrollable');

  var tbody = $(table).find('tbody');
  tbody.css('max-height', height)
  tbody.wrap('<div class="scrollable-wrapper"></div>')
  tbody.wrap('<table class="table table-striped table-bordered"></table>');

  var thead = $(table).find('thead');

  for (var i=0; i<widths.length; i++) {
    thead.find('th:eq('+i+')').width(widths[i]);
  }
}

function goto_anchor() {
  change_to_tab(window.location.hash.substring(1));

  var hash = window.location.hash;

  window.setInterval(function() {
    if (window.location.hash != hash) {
      change_to_tab(window.location.hash.substring(1));
    }

    hash = window.location.hash;
  }, 200);

  $('a[role="tab"]').on('click', function() {
    window.location.hash = '#' + $(this).attr('href').substring(5);
  });
}

var changer;

function change_to_tab(name) {
  $('a[href="#tab-'+name+'"][role="tab"]').click();
}
