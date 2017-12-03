/*
Package sns provides types to support unmarshalling generic `event *json.RawMessage` types into
SNS specific event structures.  Sparta-based SNS event listeners can unmarshall the RawMesssage
into source-specific data.  Example:

    func s3EventListener(event *json.RawMessage,
                          context *sparta.LambdaContext,
                          w http.ResponseWriter,
                          logger *logrus.Logger) {
      var lambdaEvent spartaSNS.Event
      err := json.Unmarshal([]byte(*event), &lambdaEvent)
      if err != nil {
        logger.Error("Failed to unmarshal event data: ", err.Error())
        http.Error(w, err.Error(), http.StatusInternalServerError)
      }
      for _, eachRecord := range lambdaEvent.Records {
        logger.Info("Message subject: ", eachRecord.Subject)
      }
    }
*/
package sns
