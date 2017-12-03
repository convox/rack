/*
Package s3 provides types to support unmarshalling generic `event *json.RawMessage` types into
S3 specific event structures.  Sparta-based S3 event listeners can unmarshall the RawMesssage
into source-specific data.  Example:

    func s3EventListener(event *json.RawMessage,
                          context *sparta.LambdaContext,
                          w http.ResponseWriter,
                          logger *logrus.Logger) {
      var lambdaEvent spartaS3.Event
      err := json.Unmarshal([]byte(*event), &lambdaEvent)
      if err != nil {
        logger.Error("Failed to unmarshal event data: ", err.Error())
        http.Error(w, err.Error(), http.StatusInternalServerError)
      }
      for _, eachRecord := range lambdaEvent.Records {
        switch eachRecord.EventName {
          case "ObjectCreated:Put": {...}
          default : {...}
        }
      }
    }
*/
package s3
