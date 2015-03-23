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
    $(display).append('<p>' + e.data + '</p>');
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

var tabs_enabled = true;

function activate_tabs() {
  $('a[role="tab"]').on('click', function(e) {
    e.preventDefault();

    if (tabs_enabled) {
      $('a[role="tab"]').parent().addClass('disabled');
      tabs_enabled = false;
      activate_tab($(this));
    }
  });

  $('#refresh').on('click', function() {
    refresh_tab();
  });
}

function activate_tab(tab) {
  window.location.hash = $(tab).attr('href');
  refresh_content(tab);
}

function refresh_tab() {
  refresh_content($('ul.nav-tabs li.active a'));
}

function refresh_content(tab) {
  if (!tab) {
    return;
  }

  $('#spinner').show();

  $.get($(tab).data('source'), function(data) {
    $('#spinner').hide();
    $(tab).tab('show');
    $('#tab-content').html(data);
    $('a[role="tab"]').parent().removeClass('disabled');
    $('#tab-content').find('.timeago').timeago();
    tabs_enabled = true;
  });
}

function goto_anchor(default_hash) {
  var hash = window.location.hash;

  if (hash == '') {
    hash = default_hash;
  }

  change_to_tab(hash);

  var current_hash = window.location.hash;

  window.setInterval(function() {
    if (window.location.hash != current_hash) {
      current_hash = window.location.hash;
      change_to_tab(current_hash);
    }
  }, 200);
}

var changer;

function change_to_tab(hash) {
  $('a[href="'+hash+'"][role="tab"]').click();
}

$(window).ready(function() {
  $('.timeago').timeago();

  activate_tabs();
  goto_anchor('#logs');
});
