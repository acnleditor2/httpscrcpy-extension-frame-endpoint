#include <stdbool.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#include <arpa/inet.h>
#include <unistd.h>

#include <libavcodec/avcodec.h>
#include <libswscale/swscale.h>

static bool alpha = true;
static unsigned char *video_packet = NULL;
static AVCodecParserContext *parser = NULL;
static AVCodecContext *codec_ctx = NULL;
static struct SwsContext *sws_ctx = NULL;
static AVFrame *frame = NULL;
static AVPacket *packet = NULL;
static int frame_width = 0;
static int frame_height = 0;
static unsigned char *frame_data = NULL;

static const AVCodec *get_decoder(uint32_t codec_id) {
  if (codec_id == 0x68323634) {
    return avcodec_find_decoder(AV_CODEC_ID_H264);
  } else if (codec_id == 0x68323635) {
    return avcodec_find_decoder(AV_CODEC_ID_H265);
  } else if (codec_id == 0x617631) {
    return avcodec_find_decoder(AV_CODEC_ID_AV1);
  }

  return NULL;
}

static char *read_string_8(void) {
  uint8_t length;
  if (read(STDIN_FILENO, &length, sizeof(length)) != sizeof(length)) {
    return NULL;
  }

  char *s = malloc(length + 1);
  s[length] = '\0';

  if (read(STDIN_FILENO, s, length) != length) {
    return NULL;
  }

  return s;
}

static char *read_string_32(void) {
  uint32_t length;
  if (read(STDIN_FILENO, &length, sizeof(length)) != sizeof(length)) {
    return NULL;
  }

  char *s = malloc(length + 1);
  s[length] = '\0';

  if (read(STDIN_FILENO, s, length) != length) {
    return NULL;
  }

  return s;
}

static void write_8(uint8_t i) { write(STDOUT_FILENO, &i, sizeof(i)); }
static void write_16(uint16_t i) { write(STDOUT_FILENO, &i, sizeof(i)); }
static void write_32(uint32_t i) { write(STDOUT_FILENO, &i, sizeof(i)); }

static void write_string(const char *s) {
  uint8_t length = strlen(s);
  write(STDOUT_FILENO, &length, sizeof(length));
  write(STDOUT_FILENO, s, length);
}

static int read_video_packet(void) {
  unsigned char header[12];
  if (read(STDIN_FILENO, header, sizeof(header)) != sizeof(header)) {
    return 0;
  }

  uint32_t size;
  memcpy(&size, header + 8, sizeof(size));
  size = ntohl(size);

  video_packet = malloc(size);
  if (!video_packet) {
    return 0;
  }

  unsigned char *data = video_packet;
  int len = size;

  while (len > 0) {
    int r = read(STDIN_FILENO, data, len);
    if (r < 0) {
      return 0;
    }
    data += r;
    len -= r;
  }

  return size;
}

static void extension_loop(void) {
  for (;;) {
    uint8_t type;
    if (read(STDIN_FILENO, &type, sizeof(type)) != sizeof(type)) {
      break;
    }

    switch (type) {
    case 0: {
      uint32_t query_count;
      if (read(STDIN_FILENO, &query_count, sizeof(query_count)) !=
          sizeof(query_count)) {
        return;
      }

      for (int i = 0; i < query_count; i++) {
        char *name = read_string_32();
        if (!name) {
          return;
        }
        free(name);

        char *value = read_string_32();
        if (!value) {
          return;
        }
        free(value);
      }

      uint32_t header_count;
      if (read(STDIN_FILENO, &header_count, sizeof(header_count)) !=
          sizeof(header_count)) {
        return;
      }

      for (int i = 0; i < header_count; i++) {
        char *name = read_string_32();
        if (!name) {
          return;
        }
        free(name);

        char *value = read_string_32();
        if (!value) {
          return;
        }
        free(value);
      }

      if (frame_data) {
        write_16(200);
        write_8(6);
        write_string("Cache-Control");
        write_string("no-store");
        int size = frame_width * frame_height * (alpha ? 4 : 3);
        char *s;
        write_string("Content-Length");
        asprintf(&s, "%d", size);
        write_string(s);
        free(s);
        write_string("Content-Type");
        write_string("application/octet-stream");
        write_string("Width");
        asprintf(&s, "%d", frame_width);
        write_string(s);
        free(s);
        write_string("Height");
        asprintf(&s, "%d", frame_height);
        write_string(s);
        free(s);
        write_string("Channels");
        write_string(alpha ? "4" : "3");
        write_32(size);
        write(STDOUT_FILENO, frame_data, size);
        write_32(0);
        write_8(0);
      } else {
        write_16(404);
        write_8(1);
        write_string("Cache-Control");
        write_string("no-store");
        write_32(0);
        write_8(0);
      }

      break;
    }
    case 1: {
      uint16_t port;
      if (read(STDIN_FILENO, &port, sizeof(port)) != sizeof(port)) {
        return;
      }

      char *device_name = read_string_8();
      if (!device_name) {
        return;
      }
      free(device_name);

      uint32_t codec_id;
      if (read(STDIN_FILENO, &codec_id, sizeof(codec_id)) != sizeof(codec_id)) {
        return;
      }

      uint32_t unused;
      if (read(STDIN_FILENO, &unused, sizeof(unused)) != sizeof(unused)) {
        return;
      }
      if (read(STDIN_FILENO, &unused, sizeof(unused)) != sizeof(unused)) {
        return;
      }

      av_parser_close(parser);
      avcodec_free_context(&codec_ctx);
      av_frame_free(&frame);
      av_packet_free(&packet);

      const AVCodec *codec = get_decoder(codec_id);
      if (!codec) {
        return;
      }

      parser = av_parser_init((int)codec->id);
      if (!parser) {
        return;
      }

      codec_ctx = avcodec_alloc_context3(codec);
      if (!codec_ctx) {
        return;
      }

      if (avcodec_open2(codec_ctx, codec, NULL) < 0) {
        return;
      }

      frame = av_frame_alloc();
      if (!frame) {
        return;
      }

      packet = av_packet_alloc();
      if (!packet) {
        return;
      }

      break;
    }
    case 2: {
      uint16_t port;
      if (read(STDIN_FILENO, &port, sizeof(port)) != sizeof(port)) {
        return;
      }

      int len = read_video_packet();
      if (len == 0) {
        return;
      }

      unsigned char *data = video_packet;

      while (len > 0) {
        int r =
            av_parser_parse2(parser, codec_ctx, &packet->data, &packet->size,
                             data, len, AV_NOPTS_VALUE, AV_NOPTS_VALUE, 0);

        if (r < 0) {
          return;
        }

        data += r;
        len -= r;

        if (packet->size != 0) {
          if (avcodec_send_packet(codec_ctx, packet) < 0) {
            return;
          }

          for (;;) {
            r = avcodec_receive_frame(codec_ctx, frame);
            if (r == AVERROR(EAGAIN) || r == AVERROR_EOF) {
              break;
            }
            if (r < 0) {
              return;
            }

            if (frame_width != frame->width || frame_height != frame->height) {
              frame_width = frame->width;
              frame_height = frame->height;

              if (frame_data) {
                free(frame_data);
              }

              frame_data = malloc(frame_width * frame_height * (alpha ? 4 : 3));

              if (!frame_data) {
                return;
              }
            }

            sws_ctx = sws_getCachedContext(
                sws_ctx, frame_width, frame_height, codec_ctx->pix_fmt,
                frame_width, frame_height,
                alpha ? AV_PIX_FMT_RGBA : AV_PIX_FMT_RGB24, SWS_FAST_BILINEAR,
                NULL, NULL, NULL);

            if (!sws_ctx) {
              return;
            }

            int stride = (alpha ? 4 : 3) * frame_width;

            sws_scale(sws_ctx, (const uint8_t *const *)frame->data,
                      frame->linesize, 0, frame_height, &frame_data, &stride);

            av_frame_unref(frame);
            av_packet_unref(packet);
          }
        }
      }

      free(video_packet);
      video_packet = NULL;

      break;
    }
    }
  }
}

int main(int argc, char **argv) {
  if (argc == 4 && strcmp(argv[3], "noalpha") == 0) {
    alpha = false;
  } else if (argc != 3) {
    return 0;
  }

  write_string(argv[1]);
  write_8(1);
  write_string(argv[2]);

  extension_loop();

  if (video_packet) {
    free(video_packet);
  }

  if (frame_data) {
    free(frame_data);
  }

  av_parser_close(parser);
  sws_freeContext(sws_ctx);
  avcodec_free_context(&codec_ctx);
  av_frame_free(&frame);
  av_packet_free(&packet);

  return 0;
}
