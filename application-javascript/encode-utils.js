module.exports = {
    uint8ArrayToHex(buffer) {
        return [...buffer]
            .map(x => x.toString(16).padStart(2, '0'))
            .join('');
    },

    uint64EncodeBidEndian(n) {
        let buffer = new ArrayBuffer(8);
        let view = new DataView(buffer);
        view.setBigUint64(0, n);
        return buffer;
    },

    arrayToHexString(byteArray) {
        return Array.from(byteArray, function(byte) {
            return ('0' + (byte & 0xFF).toString(16)).slice(-2);
        }).join('');
    }
};
