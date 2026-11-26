#!/usr/bin/env python3
"""Redraws the dinosaur drawings of ../ three times as fine (HDDensity).

    python3 internal/client/dinos/hd/gen.py        # writes hd/*.png

Each drawing keeps its design, palette and markings. The silhouette and the
colour regions are traced smoothly instead of in 3x3 blocks, the outline is
thinned to one image pixel and tinted by the hide under it, and detail that
only fits at this size is added: scales or feathers over the hide, a relief
that picks out the painted muscles and plates, rim light from the top left,
pointed teeth, and eyes with a pupil and a glint.
"""
import os, sys
import numpy as np
from PIL import Image
from scipy import ndimage as ndi

K = 3
HERE = os.path.dirname(os.path.abspath(__file__))

# skin: the texture laid over the hide; cell: its size in image pixels.
# eyes: boxes (x0, y0, x1, y1) of the iris in the drawing's pixels; slit: a hunter's pupil.
# teeth: the box of the mouth, where pale pixels are teeth.
# Painted by hand-written descriptions in Go instead (internal/dinopaint,
# cmd/dinopaint; see README.md here): this script leaves them alone.
PAINTED = {'rex', 'trike', 'toro', 'stego', 'thesc', 'brachio'}

CFG = {
    'rex':         dict(skin='scale', cell=(5, 4), eyes=[(87, 20, 88, 20)], slit=True, teeth=(82, 31, 105, 34)),
    'trike':       dict(skin='scale', cell=(5, 4), eyes=[(62, 32, 62, 32)]),
    'toro':        dict(skin='scale', cell=(5, 4), eyes=[(68, 36, 68, 36)]),
    'stego':       dict(skin='scale', cell=(5, 4), eyes=[(83, 39, 83, 39)]),
    'brachio':     dict(skin='scale', cell=(6, 5), eyes=[(99, 8, 99, 8)]),
    'dakota':      dict(skin='feather', eyes=[(42, 5, 42, 5)], slit=True, teeth=(38, 6, 49, 9)),
    'achero':      dict(skin='feather', eyes=[(32, 3, 32, 3)], slit=True, teeth=(29, 3, 37, 7)),
    'troodon':     dict(skin='feather', eyes=[(37, 4, 39, 6)], slit=True, teeth=(38, 7, 50, 10)),
    'ornitho':     dict(skin='feather'),
    'thesc':       dict(skin='scale', cell=(4, 3)),
    'anzu':        dict(skin='feather', eyes=[(38, 9, 38, 9)]),
    'quetzal':     dict(skin='fuzz'),
    'plesio':      dict(skin='scale', cell=(4, 3), eyes=[(54, 6, 54, 6)], slit=True, teeth=(50, 8, 62, 11)),
    'ptero_perch': dict(skin='fuzz', eyes=[(40, 11, 40, 11)]),
    'ptero_fly':   dict(skin='membrane', mid=28),
    'quetzal_fly': dict(skin='membrane', mid=46),
}

LUM = np.array([.3, .59, .11], np.float32)


def up(a, smooth=True):
    """a, K times bigger."""
    im = Image.fromarray(a.astype(np.float32), 'F')
    w, h = im.size
    return np.array(im.resize((w * K, h * K), Image.BICUBIC if smooth else Image.NEAREST))


def hashf(x, y, s=0):
    h = (x.astype(np.int64) * 374761393 + y.astype(np.int64) * 668265263 + s * 974711) & 0xffffffff
    h = ((h ^ (h >> 13)) * 1274126177) & 0xffffffff
    return ((h ^ (h >> 16)) & 0xffff) / 65535.0


def redraw(name):
    cfg = CFG[name]
    src = np.array(Image.open(os.path.join(HERE, '..', name + '.png')).convert('RGBA'))
    H, W = src.shape[:2]
    rgb = src[..., :3].astype(np.float32)
    solid = src[..., 3] >= 128
    lum = rgb @ LUM
    ink = solid & (lum < 42)
    hide = solid & ~ink
    # The colour under the ink: the nearest hide.
    idx = ndi.distance_transform_edt(~hide, return_distances=False, return_indices=True)
    fill = rgb[idx[0], idx[1]]
    idx = ndi.distance_transform_edt(~ink, return_distances=False, return_indices=True)
    inkc = rgb[idx[0], idx[1]] if ink.any() else rgb * 0 + 24

    # The silhouette and the regions of ink, traced smoothly.
    M = up(solid.astype(np.float32)) > 0.5
    INK = (up(ink.astype(np.float32)) > 0.5) & M
    # Colour: smooth, then snapped to the nearest colour the drawing has nearby.
    smooth = np.dstack([up(fill[..., c]) for c in range(3)])
    pad = np.pad(fill, ((1, 1), (1, 1), (0, 0)), mode='edge')
    yy, xx = np.mgrid[0:H * K, 0:W * K]
    sy, sx = yy // K, xx // K
    base = np.zeros((H * K, W * K, 3), np.float32)
    bestd = np.full((H * K, W * K), 1e12, np.float32)
    for dy in (-1, 0, 1):
        for dx in (-1, 0, 1):
            cand = pad[sy + 1 + dy, sx + 1 + dx]
            d = ((cand - smooth) ** 2).sum(2) + (0 if (dx == 0 and dy == 0) else 1.0)
            take = d < bestd
            base[take], bestd[take] = cand[take], d[take]
    inkcol = np.dstack([up(inkc[..., c], False) for c in range(3)])

    # How deep inside the silhouette, and inside its ink.
    D = ndi.distance_transform_edt(M)
    DI = ndi.distance_transform_edt(INK)
    gy, gx = np.gradient(ndi.gaussian_filter(M.astype(np.float32), 4.0))
    facing = gx * 0.6 + gy * 0.8  # < 0: the edge faces the light at the top left
    facing = facing / (np.abs(facing).max() + 1e-6)
    lit = facing < -0.2

    # Texture over the hide.
    b_lum = base @ LUM
    mx, mn = base.max(2), base.min(2)
    sat = (mx - mn) / np.maximum(mx, 1)
    horn = (b_lum > 178) & (sat < 0.32)  # horn, tooth, claw, pale beak: smooth
    tex = np.ones((H * K, W * K), np.float32)
    skin = cfg.get('skin')
    if skin == 'scale':
        cw, ch = cfg.get('cell', (4, 3))
        row = yy // ch
        xo = xx + (row % 2) * (cw // 2)
        cx, cy = xo % cw, yy % ch
        cell = hashf(xo // cw, row, 3)
        on = cell < 0.62
        tex[on & (cx == cw - 1) & (cy > 0)] = 0.88  # the gaps between scales, row by row
        tex[on & (cy == ch - 1) & (cx == cw - 2)] = 0.92
        tex[on & (cy == 0) & (cx <= 1)] = 1.09
        tex[(~on) & (cell > 0.9) & (cy == 1) & (cx == 1)] = 0.9  # a lone pit
    elif skin in ('feather', 'fuzz'):
        ln = 7 if skin == 'feather' else 4
        a = 0.42  # barbs lie back and a little down
        u = xx * np.cos(a) - yy * np.sin(a)
        v = xx * np.sin(a) + yy * np.cos(a)
        vr = np.floor(v / 3).astype(np.int64)
        uo = u + hashf(vr, vr * 0 + 7, 5) * 13
        seg = np.floor(uo / ln).astype(np.int64)
        h = hashf(seg, vr, 9)
        line = (np.floor(v).astype(np.int64) % 3) == 0
        tex[line & (h < 0.6)] = 0.87
        tip = (np.floor(uo).astype(np.int64) % ln) == 0
        tex[(~line) & tip & (h < 0.5)] = 1.1
    elif skin == 'membrane':
        dy_ = np.abs(yy - cfg['mid'] * K)
        wing = dy_ > 5 * K
        # Strain lines from the arm back to the trailing edge, and a sheen between them.
        u = xx + 0.35 * dy_
        lid = np.floor(u / 9).astype(np.int64)
        ph = np.floor(u).astype(np.int64) % 9
        h = hashf(lid, (yy // 6), 11)
        tex[wing & (ph == 0) & (h < 0.8)] = 0.86
        tex[wing & (ph == 1) & (h < 0.5)] = 1.07
        tex[wing & (ph >= 4) & (ph <= 5) & (hashf(lid, yy // 11, 4) < 0.35)] = 1.05
    # Relief: the painted lights and darks stand out as if lit from the top left.
    cover = np.maximum(ndi.gaussian_filter(M.astype(np.float32), 1.6), 1e-3)
    hl = ndi.gaussian_filter(np.where(M, b_lum, 0), 1.6) / cover
    emb = np.roll(hl, (-2, -2), (0, 1)) - np.roll(hl, (2, 2), (0, 1))
    relief = np.round(np.clip(emb / 60.0, -1, 1) * 3) / 3 * 0.16
    shade = np.where(horn | INK, 1.0, tex) * (1 + np.where(INK, 0, relief))
    # Form: light along the edges that face it, shade along those that do not.
    inner = M & ~INK
    shade = np.where(inner & lit & (D > 3) & (D <= 5.2), shade * 1.13, shade)
    shade = np.where(inner & (facing > 0.3) & (D > 3) & (D <= 6.2), shade * 0.86, shade)
    out = np.clip(base * shade[..., None], 0, 255)

    # Ink: lines inside stay; the outline thins to one pixel, tinted by the
    # hide, lighter where it faces the light.
    out[INK] = inkcol[INK]
    d2 = INK & (D > 1.01) & (D <= 2.01)
    d3 = INK & (D > 2.01) & (D <= 3.01)
    out[d3] = base[d3] * 0.82
    out[d2] = base[d2] * np.where(lit[d2], 0.8, 0.55)[:, None]
    o1 = INK & (D <= 1.01)
    k1 = np.where(lit[o1], 0.38, 0.62)[:, None]
    out[o1] = inkcol[o1] * k1 + base[o1] * (0.72 - k1) * 0.5
    # A silhouette edge with no ink in the drawing gets a thin tinted line all the same.
    bare = M & ~INK & (D <= 1.01)
    out[bare] = base[bare] * np.where(lit[bare], 0.74, 0.52)[:, None]

    # Teeth: pointed.
    if 'teeth' in cfg:
        x0, y0, x1, y1 = cfg['teeth']
        pale = solid & (lum > 170) & ((rgb.max(2) - rgb.min(2)) < 70)
        for x in range(x0, x1 + 1):
            y = y0
            while y <= y1:
                if not pale[y, x] or (pale[y, x - 1] and pale[y, x + 1]):
                    y += 1
                    continue
                ye = y
                while ye + 1 <= y1 and pale[ye + 1, x]:
                    ye += 1
                n = (ye - y + 1) * K
                down = not ink[y - 1, x] or ink[min(H - 1, ye + 1), x]
                gum = np.array([45, 28, 27], np.float32)
                for nb in (rgb[y, x - 1], rgb[y, x + 1]):
                    if nb @ LUM < 60:
                        gum = nb
                for j in range(n):
                    t = j if down else n - 1 - j
                    half = 1 if t < max(2, n * 0.45) else 0
                    for i in range(K):
                        px, py = x * K + i, y * K + j
                        if abs(i - 1) <= half:
                            out[py, px] = rgb[y, x] * (1.0 if i <= 1 else 0.86)
                        else:
                            out[py, px] = gum
                        M[py, px] = True
                y = ye + 1

    # Eyes: an iris with a pupil and a glint, in a dark socket.
    socket = np.array([21, 16, 14], np.float32)
    for (x0, y0, x1, y1) in cfg.get('eyes', []):
        iris = rgb[y0, x0].copy()
        big = (x1 - x0) >= 2
        g = 0 if big else 1
        X0, Y0, X1, Y1 = x0 * K - g, y0 * K - g, (x1 + 1) * K - 1 + g, (y1 + 1) * K - 1 + g
        cx, cy = (X0 + X1) / 2, (Y0 + Y1) / 2
        rx, ry = (X1 - X0 + 1) / 2, (Y1 - Y0 + 1) / 2
        for py in range(Y0 - 2, Y1 + 3):
            for px in range(X0 - 2, X1 + 3):
                if not M[py, px]:
                    continue
                e = ((px - cx) / (rx + 0.25)) ** 2 + ((py - cy) / (ry + 0.25)) ** 2
                e2 = ((px - cx) / (rx + 1.3)) ** 2 + ((py - cy) / (ry + 1.3)) ** 2
                if e <= 1:
                    out[py, px] = iris
                elif e2 <= 1:
                    out[py, px] = socket
        slit = cfg.get('slit')
        pw = 0.6 if rx < 4 else 1.1
        ph = ry - 0.4 if slit else 0.6
        for py in range(Y0, Y1 + 1):
            for px in range(X0, X1 + 1):
                if abs(px - cx) <= pw and abs(py - cy) <= ph:
                    out[py, px] = (18, 10, 6)
        gx_, gy_ = int(np.floor(cx - rx * 0.5)), int(np.floor(cy - ry * 0.55))
        glint = (255, 250, 228)
        out[gy_, gx_] = glint
        if big:
            out[gy_, gx_ + 1] = glint
            out[gy_ + 1, gx_] = glint

    alpha = np.where(M, 255, 0).astype(np.uint8)
    part = (src[..., 3] > 0) & (src[..., 3] < 255)
    if part.any():  # see-through water and foam keep their own look
        an = up(src[..., 3].astype(np.float32), False)
        pn = up(part.astype(np.float32), False) > 0.5
        srcn = np.dstack([up(rgb[..., c], False) for c in range(3)])
        out[pn] = srcn[pn]
        alpha = np.where(pn, an, alpha).astype(np.uint8)
    res = np.dstack([np.clip(np.round(out), 0, 255).astype(np.uint8), alpha])
    res[alpha == 0] = 0
    Image.fromarray(res, 'RGBA').save(os.path.join(HERE, name + '.png'), optimize=True)


if __name__ == '__main__':
    for n in sys.argv[1:] or sorted(set(CFG) - PAINTED):
        if n in PAINTED:
            sys.exit(n + ' is painted by cmd/dinopaint now: go run ./cmd/dinopaint ' + n)
        redraw(n)
        print('wrote', n)
