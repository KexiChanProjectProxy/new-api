package transformer

import (
	"testing"

	"github.com/QuantumNous/new-api/types"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestEmbeddingTransformerRoundTrip(t *testing.T) {
	raw := []byte(`{"model":"text-embedding-3-small","input":"hello world","encoding_format":"float","dimensions":512,"user":"test-user","seed":42,"temperature":0.5,"top_p":0.9,"frequency_penalty":0.1,"presence_penalty":0.2}`)

	tf := EmbeddingTransformer{}
	pivot, err := tf.Inbound(raw)
	require.NoError(t, err)
	require.Equal(t, string(types.RelayFormatEmbedding), string(pivot.RelayFormat))
	require.Equal(t, "text-embedding-3-small", pivot.Model)
	require.NotNil(t, pivot.ProviderExtensions)
	require.Equal(t, "float", pivot.ProviderExtensions["embedding_encoding_format"])
	require.Equal(t, "test-user", pivot.ProviderExtensions["embedding_user"])

	out, err := tf.Outbound(pivot)
	require.NoError(t, err)
	require.Equal(t, "text-embedding-3-small", gjson.GetBytes(out, "model").String())
	require.Equal(t, "hello world", gjson.GetBytes(out, "input").String())
	require.Equal(t, "float", gjson.GetBytes(out, "encoding_format").String())
	require.Equal(t, int64(512), gjson.GetBytes(out, "dimensions").Int())
	require.Equal(t, "test-user", gjson.GetBytes(out, "user").String())
}

func TestEmbeddingTransformerMinimalRoundTrip(t *testing.T) {
	raw := []byte(`{"model":"text-embedding-ada-002","input":["hello","world"]}`)

	tf := EmbeddingTransformer{}
	pivot, err := tf.Inbound(raw)
	require.NoError(t, err)
	require.Equal(t, string(types.RelayFormatEmbedding), string(pivot.RelayFormat))
	require.Equal(t, "text-embedding-ada-002", pivot.Model)

	out, err := tf.Outbound(pivot)
	require.NoError(t, err)
	require.Equal(t, "text-embedding-ada-002", gjson.GetBytes(out, "model").String())
	require.False(t, gjson.GetBytes(out, "encoding_format").Exists())
	require.False(t, gjson.GetBytes(out, "dimensions").Exists())
}

func TestEmbeddingTransformerInvalidJSON(t *testing.T) {
	tf := EmbeddingTransformer{}
	_, err := tf.Inbound([]byte(`not json`))
	require.Error(t, err)
}

func TestEmbeddingTransformerOutboundWrongFormat(t *testing.T) {
	tf := EmbeddingTransformer{}
	pivot := &PivotRequest{RelayFormat: types.RelayFormatOpenAI, Model: "gpt-4"}
	_, err := tf.Outbound(pivot)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unexpected relay format")
}

func TestEmbeddingTransformerOutboundNilPivot(t *testing.T) {
	tf := EmbeddingTransformer{}
	_, err := tf.Outbound(nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "nil pivot request")
}

func TestRerankTransformerRoundTrip(t *testing.T) {
	raw := []byte(`{"model":"rerank-1","query":"what is AI","documents":["doc1","doc2","doc3"],"top_n":3,"return_documents":true,"max_chunk_per_doc":100,"overlap_tokens":50}`)

	tf := RerankTransformer{}
	pivot, err := tf.Inbound(raw)
	require.NoError(t, err)
	require.Equal(t, string(types.RelayFormatRerank), string(pivot.RelayFormat))
	require.Equal(t, "rerank-1", pivot.Model)
	require.NotNil(t, pivot.ProviderExtensions)
	require.Equal(t, "what is AI", pivot.ProviderExtensions["rerank_query"])

	out, err := tf.Outbound(pivot)
	require.NoError(t, err)
	require.Equal(t, "rerank-1", gjson.GetBytes(out, "model").String())
	require.Equal(t, "what is AI", gjson.GetBytes(out, "query").String())
	require.Equal(t, int64(3), gjson.GetBytes(out, "top_n").Int())
	require.True(t, gjson.GetBytes(out, "return_documents").Bool())
}

func TestRerankTransformerMinimalRoundTrip(t *testing.T) {
	raw := []byte(`{"model":"rerank-lite","query":"test","documents":["a"]}`)

	tf := RerankTransformer{}
	pivot, err := tf.Inbound(raw)
	require.NoError(t, err)
	require.Equal(t, string(types.RelayFormatRerank), string(pivot.RelayFormat))

	out, err := tf.Outbound(pivot)
	require.NoError(t, err)
	require.Equal(t, "rerank-lite", gjson.GetBytes(out, "model").String())
	require.False(t, gjson.GetBytes(out, "top_n").Exists())
	require.False(t, gjson.GetBytes(out, "return_documents").Exists())
}

func TestRerankTransformerInvalidJSON(t *testing.T) {
	tf := RerankTransformer{}
	_, err := tf.Inbound([]byte(`{invalid`))
	require.Error(t, err)
}

func TestRerankTransformerOutboundWrongFormat(t *testing.T) {
	tf := RerankTransformer{}
	pivot := &PivotRequest{RelayFormat: types.RelayFormatEmbedding, Model: "emb"}
	_, err := tf.Outbound(pivot)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unexpected relay format")
}

func TestRerankTransformerOutboundNilPivot(t *testing.T) {
	tf := RerankTransformer{}
	_, err := tf.Outbound(nil)
	require.Error(t, err)
}

func TestImageTransformerRoundTrip(t *testing.T) {
	raw := []byte(`{"model":"dall-e-3","prompt":"a sunset","n":2,"size":"1024x1024","quality":"hd","response_format":"b64_json"}`)

	tf := ImageTransformer{}
	pivot, err := tf.Inbound(raw)
	require.NoError(t, err)
	require.Equal(t, string(types.RelayFormatOpenAIImage), string(pivot.RelayFormat))
	require.Equal(t, "dall-e-3", pivot.Model)
	require.NotNil(t, pivot.ProviderExtensions)
	require.Equal(t, "a sunset", pivot.ProviderExtensions["image_prompt"])
	require.Equal(t, "1024x1024", pivot.ProviderExtensions["image_size"])
	require.Equal(t, "hd", pivot.ProviderExtensions["image_quality"])

	out, err := tf.Outbound(pivot)
	require.NoError(t, err)
	require.Equal(t, "dall-e-3", gjson.GetBytes(out, "model").String())
	require.Equal(t, "a sunset", gjson.GetBytes(out, "prompt").String())
	require.Equal(t, int64(2), gjson.GetBytes(out, "n").Int())
	require.Equal(t, "1024x1024", gjson.GetBytes(out, "size").String())
	require.Equal(t, "hd", gjson.GetBytes(out, "quality").String())
	require.Equal(t, "b64_json", gjson.GetBytes(out, "response_format").String())
}

func TestImageTransformerMinimalRoundTrip(t *testing.T) {
	raw := []byte(`{"model":"dall-e-2","prompt":"a cat"}`)

	tf := ImageTransformer{}
	pivot, err := tf.Inbound(raw)
	require.NoError(t, err)
	require.Equal(t, string(types.RelayFormatOpenAIImage), string(pivot.RelayFormat))

	out, err := tf.Outbound(pivot)
	require.NoError(t, err)
	require.Equal(t, "dall-e-2", gjson.GetBytes(out, "model").String())
	require.Equal(t, "a cat", gjson.GetBytes(out, "prompt").String())
	require.False(t, gjson.GetBytes(out, "n").Exists())
	require.False(t, gjson.GetBytes(out, "size").Exists())
}

func TestImageTransformerInvalidJSON(t *testing.T) {
	tf := ImageTransformer{}
	_, err := tf.Inbound([]byte(`not-json`))
	require.Error(t, err)
}

func TestImageTransformerOutboundWrongFormat(t *testing.T) {
	tf := ImageTransformer{}
	pivot := &PivotRequest{RelayFormat: types.RelayFormatOpenAI, Model: "gpt-4"}
	_, err := tf.Outbound(pivot)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unexpected relay format")
}

func TestImageTransformerOutboundNilPivot(t *testing.T) {
	tf := ImageTransformer{}
	_, err := tf.Outbound(nil)
	require.Error(t, err)
}

func TestAudioTransformerRoundTrip(t *testing.T) {
	raw := []byte(`{"model":"tts-1","input":"Hello, world!","voice":"alloy","instructions":"Speak cheerfully","response_format":"mp3","speed":1.25,"stream_format":"sse"}`)

	tf := AudioTransformer{}
	pivot, err := tf.Inbound(raw)
	require.NoError(t, err)
	require.Equal(t, string(types.RelayFormatOpenAIAudio), string(pivot.RelayFormat))
	require.Equal(t, "tts-1", pivot.Model)
	require.NotNil(t, pivot.ProviderExtensions)
	require.Equal(t, "Hello, world!", pivot.ProviderExtensions["audio_input"])
	require.Equal(t, "alloy", pivot.ProviderExtensions["audio_voice"])
	require.Equal(t, "Speak cheerfully", pivot.ProviderExtensions["audio_instructions"])
	require.Equal(t, "mp3", pivot.ProviderExtensions["audio_response_format"])
	require.Equal(t, "sse", pivot.ProviderExtensions["audio_stream_format"])

	out, err := tf.Outbound(pivot)
	require.NoError(t, err)
	require.Equal(t, "tts-1", gjson.GetBytes(out, "model").String())
	require.Equal(t, "Hello, world!", gjson.GetBytes(out, "input").String())
	require.Equal(t, "alloy", gjson.GetBytes(out, "voice").String())
	require.Equal(t, "Speak cheerfully", gjson.GetBytes(out, "instructions").String())
	require.Equal(t, "mp3", gjson.GetBytes(out, "response_format").String())
	require.Equal(t, 1.25, gjson.GetBytes(out, "speed").Float())
	require.Equal(t, "sse", gjson.GetBytes(out, "stream_format").String())
}

func TestAudioTransformerMinimalRoundTrip(t *testing.T) {
	raw := []byte(`{"model":"tts-1-hd","input":"Hi","voice":"echo"}`)

	tf := AudioTransformer{}
	pivot, err := tf.Inbound(raw)
	require.NoError(t, err)
	require.Equal(t, string(types.RelayFormatOpenAIAudio), string(pivot.RelayFormat))

	out, err := tf.Outbound(pivot)
	require.NoError(t, err)
	require.Equal(t, "tts-1-hd", gjson.GetBytes(out, "model").String())
	require.Equal(t, "Hi", gjson.GetBytes(out, "input").String())
	require.Equal(t, "echo", gjson.GetBytes(out, "voice").String())
	require.False(t, gjson.GetBytes(out, "speed").Exists())
	require.False(t, gjson.GetBytes(out, "stream_format").Exists())
}

func TestAudioTransformerInvalidJSON(t *testing.T) {
	tf := AudioTransformer{}
	_, err := tf.Inbound([]byte(`{bad json`))
	require.Error(t, err)
}

func TestAudioTransformerOutboundWrongFormat(t *testing.T) {
	tf := AudioTransformer{}
	pivot := &PivotRequest{RelayFormat: types.RelayFormatEmbedding, Model: "emb"}
	_, err := tf.Outbound(pivot)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unexpected relay format")
}

func TestAudioTransformerOutboundNilPivot(t *testing.T) {
	tf := AudioTransformer{}
	_, err := tf.Outbound(nil)
	require.Error(t, err)
}

func TestNonChatNoopResponseTransformerErrors(t *testing.T) {
	tf := NonChatNoopResponseTransformer{}
	_, err := tf.InboundResponse([]byte(`{}`))
	require.Error(t, err)
	require.Contains(t, err.Error(), "non-chat formats do not support")

	_, err = tf.OutboundResponse(nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "non-chat formats do not support")
}

func TestNonChatNoopStreamTransformerErrors(t *testing.T) {
	tf := NonChatNoopStreamTransformer{}
	_, err := tf.InboundStream([]byte(`{}`))
	require.Error(t, err)
	require.Contains(t, err.Error(), "non-chat formats do not support")

	_, err = tf.OutboundStream(nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "non-chat formats do not support")
}

func TestNonChatTransformersRegisteredInGlobalRegistry(t *testing.T) {
	formats := []types.RelayFormat{
		types.RelayFormatEmbedding,
		types.RelayFormatRerank,
		types.RelayFormatOpenAIImage,
		types.RelayFormatOpenAIAudio,
	}
	for _, format := range formats {
		tf, ok := GetTransformer(format)
		require.True(t, ok, "expected transformer for format %q to be registered", format)
		require.NotNil(t, tf)

		respTf, ok := GetResponseTransformer(format)
		require.True(t, ok, "expected response transformer for format %q to be registered", format)
		require.NotNil(t, respTf)

		streamTf, ok := GetStreamTransformer(format)
		require.True(t, ok, "expected stream transformer for format %q to be registered", format)
		require.NotNil(t, streamTf)
	}
}
