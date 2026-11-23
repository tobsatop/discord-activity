import { test } from 'node:test';
import assert from 'node:assert/strict';
import { fit, parseTrack, toActivity, type Track } from '../activity.ts';

const SONG: Track = {
  videoId: 'dQw4w9WgXcQ',
  title: 'Song Title',
  artist: 'Some Artist',
  album: 'Some Album',
  artworkUrl: 'https://lh3.googleusercontent.com/abc=w544-h544-l90-rj',
  startMs: 1_700_000_000_000,
  endMs: 1_700_000_200_000,
  playing: true,
};

test('a playing song becomes a "Listening to" activity', () => {
  assert.deepEqual(toActivity(SONG), {
    type: 2,
    status_display_type: 2,
    details: 'Song Title',
    details_url: 'https://music.youtube.com/watch?v=dQw4w9WgXcQ',
    state: 'Some Artist',
    timestamps: { start: 1_700_000_000_000, end: 1_700_000_200_000 },
    assets: { large_image: SONG.artworkUrl, large_text: 'Some Album' },
    buttons: [{ label: 'Listen on YouTube Music', url: 'https://music.youtube.com/watch?v=dQw4w9WgXcQ' }],
  });
});

test('paused or missing song clears the activity', () => {
  assert.equal(toActivity({ ...SONG, playing: false }), null);
  assert.equal(toActivity(null), null);
});

test('basic mode leaves out the newer optional fields', () => {
  const activity = toActivity(SONG, { basic: true });
  assert.equal(activity?.status_display_type, undefined);
  assert.equal(activity?.details_url, undefined);
  assert.equal(activity?.details, 'Song Title');
});

test('unknown length (live) shows elapsed time only', () => {
  assert.deepEqual(toActivity({ ...SONG, endMs: 0 })?.timestamps, { start: SONG.startMs });
});

test('artwork that is too long or not https falls back to the video thumbnail', () => {
  const thumb = 'https://i.ytimg.com/vi/dQw4w9WgXcQ/hqdefault.jpg';
  assert.equal(toActivity({ ...SONG, artworkUrl: `https://x.com/${'a'.repeat(300)}` })?.assets?.large_image, thumb);
  assert.equal(toActivity({ ...SONG, artworkUrl: 'http://insecure/img.jpg' })?.assets?.large_image, thumb);
  assert.equal(toActivity({ ...SONG, artworkUrl: '', videoId: '' })?.assets, undefined);
});

test('no video id means no link and no button', () => {
  const activity = toActivity({ ...SONG, videoId: '' });
  assert.equal(activity?.details_url, undefined);
  assert.equal(activity?.buttons, undefined);
});

test('fit keeps text within 2..128 chars without splitting emoji', () => {
  assert.equal(fit('A'), 'A⠀');
  assert.equal(fit('  many   spaces  '), 'many spaces');
  const long = fit('😀'.repeat(100));
  assert.ok(long.length <= 128);
  assert.ok(long.endsWith('…'));
  assert.ok(!/[\uD800-\uDBFF]…$/.test(long), 'no lone surrogate before the ellipsis');
  assert.equal(fit('ก'.repeat(200)).length, 128);
});

test('parseTrack rejects junk and cleans fields', () => {
  assert.equal(parseTrack(null), null);
  assert.equal(parseTrack('song'), null);
  assert.equal(parseTrack({ title: '   ' }), null);
  const parsed = parseTrack({ title: ' Hi ', videoId: 'bad id', startMs: 'x', playing: 'yes' });
  assert.deepEqual(parsed, {
    videoId: '',
    title: 'Hi',
    artist: '',
    album: '',
    artworkUrl: '',
    startMs: 0,
    endMs: 0,
    playing: false,
  });
});
