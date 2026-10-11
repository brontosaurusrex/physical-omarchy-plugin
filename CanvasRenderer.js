.pragma library

function roundedRect(ctx, x, y, w, h, radius) {
    var r = Math.max(0, Math.min(Number(radius) || 0, w / 2, h / 2));
    ctx.moveTo(x + r, y);
    ctx.lineTo(x + w - r, y); ctx.quadraticCurveTo(x + w, y, x + w, y + r);
    ctx.lineTo(x + w, y + h - r); ctx.quadraticCurveTo(x + w, y + h, x + w - r, y + h);
    ctx.lineTo(x + r, y + h); ctx.quadraticCurveTo(x, y + h, x, y + h - r);
    ctx.lineTo(x, y + r); ctx.quadraticCurveTo(x, y, x + r, y);
    ctx.closePath();
}

function paint(ctx, ops, sx, sy, theme, fontFamily, clear) {
    if (clear) { ctx.reset(); ctx.clearRect(0, 0, ctx.canvas.width, ctx.canvas.height); ctx.scale(sx, sy); }
    for (var i = 0; i < ops.length; ++i) {
        var op = ops[i], name = op[0];
        switch (name) {
        case "set":
            var value = op[2];
            if (op[1] === "font") value = String(value).replace(/GameFont,\s*monospace/g, '"' + fontFamily + '"');
            ctx[op[1]] = value;
            break;
        case "roundRect": roundedRect(ctx, op[1], op[2], op[3], op[4], op[5]); break;
        case "setTransform": ctx.setTransform(op[1] * sx, op[2] * sy, op[3] * sx, op[4] * sy, op[5] * sx, op[6] * sy); break;
        case "clearRect": break; // Each layer is cleared once before its command stream.
        case "save": ctx.save(); break;
        case "restore": ctx.restore(); break;
        case "beginPath": ctx.beginPath(); break;
        case "closePath": ctx.closePath(); break;
        case "fill": ctx.fill(); break;
        case "stroke": ctx.stroke(); break;
        case "moveTo": ctx.moveTo(op[1], op[2]); break;
        case "lineTo": ctx.lineTo(op[1], op[2]); break;
        case "quadraticCurveTo": ctx.quadraticCurveTo(op[1], op[2], op[3], op[4]); break;
        case "arc": ctx.arc(op[1], op[2], op[3], op[4], op[5], op.length > 6 ? op[6] : false); break;
        case "translate": ctx.translate(op[1], op[2]); break;
        case "rotate": ctx.rotate(op[1]); break;
        case "fillRect": ctx.fillRect(op[1], op[2], op[3], op[4]); break;
        case "fillText": ctx.fillText(op[1], op[2], op[3]); break;
        case "setLineDash": ctx.setLineDash(op[1]); break;
        default: throw new Error("Unsupported native drawing command: " + name);
        }
    }
}

function layers(ops) {
    for (var i = 0; i < ops.length; ++i) {
        if (ops[i][0] === "cache") return { before: ops.slice(0, i), cache: ops[i][1], after: ops.slice(i + 1) };
    }
    return { before: [], cache: -1, after: ops };
}
