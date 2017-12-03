/*
Package kinesis provides types to support unmarshalling generic `event *json.RawMessage` types into
Kinesis specific event structures.  Sparta-based Kinesis event listeners can unmarshall the RawMesssage
into source-specific data.  Example:

    func s3EventListener(event *json.RawMessage,
                          context *sparta.LambdaContext,
                          w http.ResponseWriter,
                          logger *logrus.Logger) {
      var lambdaEvent spartaKinesis.Event
      err := json.Unmarshal([]byte(*event), &lambdaEvent)
      if err != nil {
        logger.Error("Failed to unmarshal event data: ", err.Error())
        http.Error(w, err.Error(), http.StatusInternalServerError)
      }
      for _, eachRecord := range lambdaEvent.Records {
        logger.Info("Kinesis event:", eachRecord.EventName)
      }
    }
*/
package kinesis
