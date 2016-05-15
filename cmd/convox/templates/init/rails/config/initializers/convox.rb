STDOUT.sync = true

Rails.application.config.before_initialize do |app|
  app.config.logger = Logger.new(STDOUT)
end
