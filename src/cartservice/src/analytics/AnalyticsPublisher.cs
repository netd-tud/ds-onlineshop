using System;
using System.Text.Json;
using System.Text.Json.Serialization;
using Confluent.Kafka;
using Microsoft.Extensions.Logging;

namespace cartservice.analytics
{
    public class ProductEvent
    {
        [JsonPropertyName("event_id")]
        public string EventId { get; set; }

        [JsonPropertyName("event_time")]
        public DateTime EventTime { get; set; }

        [JsonPropertyName("event_type")]
        public string EventType { get; set; }

        [JsonPropertyName("sku")]
        public string Sku { get; set; }

        [JsonPropertyName("qty")]
        [JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingDefault)]
        public int Qty { get; set; }

        [JsonPropertyName("price")]
        [JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingDefault)]
        public double Price { get; set; }

        [JsonPropertyName("session_id")]
        public string SessionId { get; set; }

        [JsonPropertyName("producer")]
        public string Producer { get; set; }
    }

    [JsonSourceGenerationOptions(WriteIndented = false)]
    [JsonSerializable(typeof(ProductEvent))]
    internal partial class ProductEventJsonContext : JsonSerializerContext
    {
    }

    public class AnalyticsPublisher : IDisposable
    {
        private readonly IProducer<string, string> _producer;
        private readonly string _producerName;
        private readonly string _topic;
        private readonly ILogger<AnalyticsPublisher> _logger;

        public AnalyticsPublisher(ILogger<AnalyticsPublisher> logger)
        {
            _logger = logger;
            _producerName = "cartservice";

            var brokers = Environment.GetEnvironmentVariable("KAFKA_BOOTSTRAP_SERVERS")
                          ?? "analytics-kafka-kafka-bootstrap.kafka.svc.cluster.local:9092";
            _topic = Environment.GetEnvironmentVariable("KAFKA_PRODUCT_EVENTS_TOPIC")
                     ?? "product-events";

            var config = new ProducerConfig
            {
                BootstrapServers = brokers,
                Acks = Acks.Leader,
                LingerMs = 50,
                Partitioner = Partitioner.Consistent
            };

            _producer = new ProducerBuilder<string, string>(config).Build();
        }

        public void Publish(ProductEvent evt)
        {
            evt.EventId = Guid.NewGuid().ToString();
            evt.EventTime = DateTime.UtcNow;
            evt.Producer = _producerName;

            var payload = JsonSerializer.Serialize(evt, ProductEventJsonContext.Default.ProductEvent);
            var message = new Message<string, string> { Key = evt.Sku, Value = payload };

            _producer.Produce(_topic, message, deliveryReport =>
            {
                if (deliveryReport.Error.IsError)
                {
                    _logger.LogError($"analytics: publish {evt.EventType} for {evt.Sku} failed: {deliveryReport.Error.Reason}");
                }
            });
        }

        public void Dispose()
        {
            _producer.Flush(TimeSpan.FromSeconds(2));
            _producer.Dispose();
        }
    }
}


