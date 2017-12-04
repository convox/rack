package ses

/*{ Records:
  [ { eventSource: 'aws:ses',
      eventVersion: '1.0',
      ses:
       { mail:
          {
            timestamp: '2016-01-15T13:47:34.435Z',
            source: 'user@domain.com',
            messageId: 'qj5icnbmpuh22t8b5p4o50854r3qiop2nhdcjto1',
            destination: [ 'sombody_special@gosparta.io' ],
            headersTruncated: false,
            headers:
             [
              { name: 'Return-Path', value: '<user@domain.com>' },
               { name: 'X-Received',
                 value: 'by 10.66.150.37 with SMTP id uf5mr15354573pab.30.1452865653558; Fri, 15 Jan 2016 05:47:33 -0800 (PST)' },
               { name: 'Return-Path', value: '<user@domain.com>' },
               { name: 'Received',
                 value: 'from [192.150.23.157] (c-50-135-43-1.hsd1.wa.comcast.net. [8.8.8.8]) by smtp.gmail.com with ESMTPSA id t67sm15840589pfa.14.2016.01.15.05.47.32 for <sombody_special@gosparta.io> (version=TLS1 cipher=ECDHE-RSA-AES128-SHA bits=128/128); Fri, 15 Jan 2016 05:47:32 -0800 (PST)' },
               { name: 'From', value: 'User <user@domain.com>' },
               { name: 'Content-Type', value: 'text/plain; charset=us-ascii' },
               { name: 'Content-Transfer-Encoding', value: '7bit' },
               { name: 'Subject', value: 'Test subject content' },
               { name: 'Message-Id',
                 value: '<DEADBEEF-743C-44DB-8FAB-C65989FE7F80@gmail.com>' },
               { name: 'Date', value: 'Fri, 15 Jan 2016 05:47:30 -0800' },
               { name: 'To', value: 'sombody_special@gosparta.io' },
               { name: 'Mime-Version',
                 value: '1.0 (Mac OS X Mail 9.2 \\(3112\\))' },
               { name: 'X-Mailer', value: 'Apple Mail (2.3112)' }
               ],
            commonHeaders:
             {
              returnPath: 'user@domain.com',
               from: [ 'User <user@domain.com>' ],
               date: 'Fri, 15 Jan 2016 05:47:30 -0800',
               to: [ 'sombody_special@gosparta.io' ],
               messageId: '<DEADBEEF-743C-44DB-8FAB-C65989FE7F80@gmail.com>',
               subject: 'Test subject content' }
             },
         receipt:
          { timestamp: '2016-01-15T13:47:34.435Z',
            processingTimeMillis: 1044,
            recipients: [ 'sombody_special@gosparta.io' ],
            spamVerdict: { status: 'PASS' },
            virusVerdict: { status: 'PASS' },
            spfVerdict: { status: 'PASS' },
            dkimVerdict: { status: 'PASS' },
            action:
             { type: 'Lambda',
               functionArn: 'arn:aws:lambda:us-west-2:123412341234:function:MyApp-Lambda1dee56d1dfad99063961760c25-FFXF3GZDN6V4',
               invocationType: 'Event' } } } } ] }
*/

// Verdict result data
type Verdict struct {
	Status string `json:"status"`
}

// Receipt mail data
type Receipt struct {
	Timestamp            string   `json:"timestamp"`
	ProcessingTimeMillis uint64   `json:"processingTimeMillis"`
	Receipients          []string `json:"recipients"`
	SpamVerdict          Verdict  `json:"spamVerdict"`
	VirusVerdict         Verdict  `json:"virusVerdict"`
	SPFVerdict           Verdict  `json:"spfVerdict"`
	DKIMVerdict          Verdict  `json:"dkimVerdict"`
	// Skip the actions...
}

// CommonHeaders mail data
type CommonHeaders struct {
	ReturnPath string   `json:"returnPath"`
	From       []string `json:"from"`
	Date       string   `json:"date"`
	To         []string `json:"to"`
	MessageID  string   `json:"messageId"`
	Subject    string   `json:"subject"`
}

// Header mail data
type Header struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// Mail event data
type Mail struct {
	Timestamp        string        `json:"timestamp"`
	Source           string        `json:"source"`
	MessageID        string        `json:"messageId"`
	Destination      []string      `json:"destination"`
	HeadersTruncated bool          `json:"headersTruncated"`
	Headers          []Header      `json:"headers"`
	CommonHeaders    CommonHeaders `json:"commonHeaders"`
	Receipt          Receipt       `json:"receipt"`
}

// SES event information
type SES struct {
	Mail Mail `json:"mail"`
}

// EventRecord event data
type EventRecord struct {
	Source  string `json:"eventSource"`
	Version string `json:"eventVersion"`
	SES     SES    `json:"ses"`
}

// Event data
type Event struct {
	Records []EventRecord
}
