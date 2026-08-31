package network.sudharma.wallet.chain.sudharma

import org.bouncycastle.asn1.x9.ECNamedCurveTable
import org.bouncycastle.crypto.params.ECDomainParameters
import org.bouncycastle.crypto.params.ECPrivateKeyParameters
import org.bouncycastle.crypto.params.ECPublicKeyParameters
import org.bouncycastle.crypto.params.ParametersWithRandom
import org.bouncycastle.crypto.signers.ECDSASigner
import java.math.BigInteger
import java.security.MessageDigest
import java.security.SecureRandom

object SudharmaCrypto {
    private val params = requireNotNull(ECNamedCurveTable.getByName("secp256r1"))
    private val domain = ECDomainParameters(params.curve, params.g, params.n, params.h)
    private val random = SecureRandom()

    data class KeyMaterial(val privateScalar: BigInteger, val publicKey: ByteArray) {
        override fun equals(other: Any?): Boolean = other is KeyMaterial &&
            privateScalar == other.privateScalar && publicKey.contentEquals(other.publicKey)
        override fun hashCode(): Int = 31 * privateScalar.hashCode() + publicKey.contentHashCode()
    }

    fun keyFromPrivateScalar(privateScalar: BigInteger): KeyMaterial {
        require(privateScalar.signum() > 0 && privateScalar < params.n) { "invalid private scalar" }
        val publicPoint = params.g.multiply(privateScalar).normalize()
        return KeyMaterial(privateScalar, publicPoint.getEncoded(false))
    }

    fun addressFromPublicKey(publicKey: ByteArray): String {
        require(publicKey.size == 65 && publicKey.first() == 0x04.toByte()) { "invalid public key" }
        val hash = sha256(publicKey)
        return hash.copyOfRange(0, 20).toHex()
    }

    fun sign(privateScalar: BigInteger, message: ByteArray): ByteArray {
        val key = keyFromPrivateScalar(privateScalar)
        val signer = ECDSASigner()
        signer.init(
            true,
            ParametersWithRandom(ECPrivateKeyParameters(key.privateScalar, domain), random),
        )
        val (r, s) = signer.generateSignature(sha256(message))
        return r.toFixed32() + s.toFixed32()
    }

    fun verify(publicKey: ByteArray, message: ByteArray, signature: ByteArray): Boolean {
        if (signature.size != 64) return false
        return runCatching {
            val point = params.curve.decodePoint(publicKey).normalize()
            val signer = ECDSASigner()
            signer.init(false, ECPublicKeyParameters(point, domain))
            val r = BigInteger(1, signature.copyOfRange(0, 32))
            val s = BigInteger(1, signature.copyOfRange(32, 64))
            signer.verifySignature(sha256(message), r, s)
        }.getOrDefault(false)
    }

    fun sha256(data: ByteArray): ByteArray = MessageDigest.getInstance("SHA-256").digest(data)

    private fun BigInteger.toFixed32(): ByteArray {
        val raw = toByteArray().let { if (it.size == 33 && it[0] == 0.toByte()) it.copyOfRange(1, 33) else it }
        require(raw.size <= 32) { "signature integer too large" }
        return ByteArray(32 - raw.size) + raw
    }

    private fun ByteArray.toHex(): String = joinToString("") { "%02x".format(it.toInt() and 0xff) }
}
