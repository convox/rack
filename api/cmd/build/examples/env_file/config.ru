require 'json'

run lambda { |env| [200, {'Content-Type'=>'application/json'}, StringIO.new(JSON.pretty_generate(ENV.to_h))] }
