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

function refresh_tab(cb) {
  refresh_content($('ul.nav-tabs li.active a'), cb);
}

function refresh_content(tab, cb) {
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

    apply_tab_events();

    if (cb !== undefined) {
      cb(tab);
    }
  });
}

function apply_tab_events() {
  $('#environment-add').on('click', function() {
    var cluster = $(this).data('cluster');
    var app = $(this).data('app');
    var name = $('.app-environment tfoot input[name="name"]').val();
    var value = $('.app-environment tfoot input[name="value"]').val();

    $.post('/apps/' + app + '/environment/' + name, { value:value }, function() {
      refresh_tab();
    });
  });

  $('.environment-delete').on('click', function() {
    var cluster = $(this).data('cluster');
    var app = $(this).data('app');
    var name = $(this).data('name');

    $.ajax({ method:"DELETE", url:'/apps/' + app + '/environment/' + name }).done(function() {
      refresh_tab();
    });
  });

  $('#environment-raw').on('click', function() {
    $('#environment-basic-content').hide();
    $('#environment-raw-content').show();
  });

  $('#environment-raw-cancel').on('click', function() {
    $('#environment-raw-content').hide();
    $('#environment-basic-content').show();
  });

  $('#environment-raw-save').on('click', function() {
    var cluster = $(this).data('cluster');
    var app = $(this).data('app');

    $.post('/apps/' + app + '/environment', $('#environment-content').val(), function() {
      refresh_tab();
    });
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

function change_to_tab(hash, cb) {
  refresh_content($('a[href="'+hash+'"][role="tab"]'), cb);
}

$(window).ready(function() {
  $('.timeago').timeago();

  activate_tabs();

  $(document).on("click", "#tab-content .pager a", function(e) {
    e.preventDefault()

    $.get($(e.target).attr("href"), function(data) {
      $('#tab-content').html(data)
    })
  })

});
